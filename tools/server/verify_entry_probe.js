#!/usr/bin/env node
// verify_entry_probe.js — TEMPORARY verification probe for the Single-Entry refactor.
// Confirms: (1) zero console/page errors, (2) modules previously loaded as classic
// <script src="js/*"> tags actually EVALUATED via app.js imports, (3) the corrupted
// index.html tail no longer leaks JS as visible page text, (4) index.html contains
// no first-party script tags other than app.js.
//
// Usage: node tools/server/verify_entry_probe.js [--url http://localhost:8090/]

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
  const cdpPort = 9444;

  const chrome = resolveChrome();
  if (!chrome) { console.log('[PROBE] No browser found — skipped.'); process.exit(0); }

  const profile = fs.mkdtempSync(path.join(os.tmpdir(), 'vbt-probe-'));
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
      if (p.type === 'error') errors.push('console: ' + (p.args || []).map((a) => a.value || a.description || '').join(' ').slice(0, 180));
    });
    cdp.on('Runtime.exceptionThrown', (p) => {
      const d = p.exceptionDetails || {};
      errors.push('pageerror: ' + ((d.exception && (d.exception.description || d.exception.value)) || d.text || '').slice(0, 180));
    });

    await cdp.send('Runtime.enable');
    await cdp.send('Page.enable');
    await cdp.send('Page.navigate', { url: target });
    // WAIT FOR THE ENGINE, not for a guess. The slow part of boot is the WASM engine, and every
    // assertion below assumes it finished — a fixed sleep is both slower than necessary and
    // unreliable on a cold cache. Poll readiness up to `--wait <ms>` (default 20000) instead.
    const waitArg = argv.indexOf('--wait');
    const waitMs = waitArg >= 0 ? Number(argv[waitArg + 1]) || 20000 : 20000;
    const readyExpr = `(() => typeof window.GetGameState === 'function' && !!document.body.classList.contains('ready'))()`;
    const readyDeadline = Date.now() + waitMs;
    let ready = false;
    while (Date.now() < readyDeadline) {
      const r = await cdp.send('Runtime.evaluate', { expression: readyExpr, returnByValue: true, awaitPromise: true });
      if (r.result && r.result.value === true) { ready = true; break; }
      await new Promise((r2) => setTimeout(r2, 250));
    }
    if (!ready) console.log('[PROBE] WARNING: engine readiness was not observed within ' + waitMs + 'ms');

    const expr = `(() => {
      const out = {};
      const has = (n) => typeof window[n] === 'function';
      out.globals = {
        WasmWalletBridge: !!(window.WasmWalletBridge && window.WasmWalletBridge.signMarketAction),
        UnifiedAlertSystem: !!(window.UnifiedAlertSystem && window.UnifiedAlertSystem.showAlert),
        closeConstellationHub: has('closeConstellationHub'),
        openConstellationHub: has('openConstellationHub'),
        open3DWorld: has('open3DWorld'),
        BountyTracker: !!(window.BountyTracker),
        openWorldDashboard: has('openWorldDashboard'),
        openPortfolio: has('openPortfolio'),
        openPlayerProfile: has('openPlayerProfile'), // deprecated alias
        openSeasonalEvents: has('openSeasonalEvents'),
        openAchievements: has('openAchievements'),
        openUnderworld: has('openUnderworld'),
        enter3DWorld: has('enter3DWorld'),
        GetGameState: has('GetGameState')
      };
      out.leakedJsText = /openConstellationHub\\(\\)|function closeConstellationHub|window\\.location\\.href = .\\/world/.test(document.body.innerText || '');
      out.scripts = Array.from(document.querySelectorAll('script[src]'))
        .map(s => s.getAttribute('src'))
        .filter(s => s && !/^https?:\\/\\//.test(s) && !/wasm_exec/.test(s) && !/^\\/vendor\\//.test(s) && s !== 'app.js');
      out.ready = document.body.classList.contains('ready');
      return out;
    })()`;

    const r = await cdp.send('Runtime.evaluate', { expression: expr, returnByValue: true, awaitPromise: true });
    const v = r.result.value;

    let pass = 0, fail = 0;
    const check = (name, ok, detail) => {
      if (ok) { pass++; console.log(`  PASS  ${name}  (${detail})`); }
      else { fail++; console.log(`  FAIL  ${name}  (${detail})`); }
    };

    console.log('\n[PROBE] Single-Entry verification — ' + target);
    console.log('-'.repeat(60));
    for (const [k, ok] of Object.entries(v.globals)) check('global.' + k, ok, ok);
    check('no_leaked_js_text_in_body', v.leakedJsText === false, v.leakedJsText);
    check('first_party_script_tags_empty', v.scripts.length === 0, JSON.stringify(v.scripts));
    check('body_ready_class', v.ready === true, v.ready);
    check('zero_console_or_page_errors', errors.length === 0, errors.length ? errors.slice(0, 5).join(' | ') : '0');
    // --- NOTE VOCABULARY: the single owner of every on-chain note prefix -----
    // A note prefix is the ONE thing a client and the server must agree on
    // EXACTLY: if they disagree, nothing throws — the server simply never matches
    // the payment, which looks like "payments are broken" rather than "the client
    // sent the wrong prefix". So the prefix is FETCHED, never re-typed. These
    // assertions pin the served contract, the exact-match rule, and the client
    // reader's fail-closed behaviour.
    const vocabRt = await cdp.send('Runtime.evaluate', {
      expression: `(async () => {
        const out = {};
        const sleep = (ms) => new Promise(r => setTimeout(r, ms));
        // One-shot 429 retry, the repo-wide convention for a refused read.
        const json = async (url, opts) => {
          for (let attempt = 0; attempt < 2; attempt++) {
            const res = await fetch(url, opts);
            if (res.status === 429 && attempt === 0) { await sleep(500); continue; }
            return { status: res.status, body: await res.json().catch(() => null) };
          }
          return { status: 429, body: null };
        };
        try {
          const g = await json('/api/notes/vocabulary');
          out.status = g.status;
          const b = g.body || {};
          out.total = (b.counts || {}).total || 0;
          out.moneyDoor = (b.counts || {}).money_door || 0;
          out.checkpoint = (b.counts || {}).checkpoint || 0;
          out.auditLog = (b.counts || {}).audit_log || 0;
          const rules = b.rules || {};
          out.rulesOk = rules.client_may_name_money_door_only === true &&
                        rules.prefixes_are_exact_match === true &&
                        rules.empty_prefix_is_refused === true;
          const purposes = Array.isArray(b.purposes) ? b.purposes : [];
          out.doorPrefixes = purposes.filter(p => p.scope === 'money_door').map(p => p.prefix);
          // EXACT MATCH: no declared prefix may be a PROPER prefix of another,
          // otherwise one payment could satisfy two purposes.
          const all = purposes.map(p => p.prefix).filter(Boolean);
          out.ambiguous = all.filter(a => all.some(o => o !== a && o.indexOf(a) === 0));
          // The client's read-only reader: fetched, never re-typed.
          out.hasModule = !!(window.NoteVocab && window.NoteVocab.load);
          if (out.hasModule) {
            await window.NoteVocab.load(true);
            out.ready = window.NoteVocab.ready() === true;
            out.built = window.NoteVocab.buildNote('courthouse_fine', 'tx-1');
            out.unknownKeyRefused = window.NoteVocab.prefix('__not_a_purpose__') === '';
            out.emptyPartRefused = window.NoteVocab.buildNote('courthouse_fine', '') === '';
            out.doorCount = window.NoteVocab.moneyDoors().length;
          }
          // The client must not SPELL a prefix: criminality.js (the courthouse +
          // bail surface) is fetched and checked against the served money doors.
          const js = await fetch('/js/criminality.js');
          const text = js.ok ? await js.text() : '';
          out.jsStatus = js.status;
          out.spelledPrefixes = out.doorPrefixes.filter(p => text.indexOf(p) !== -1);
          const post = await fetch('/api/notes/vocabulary', { method: 'POST' });
          out.postStatus = post.status;
        } catch (e) { out.error = e.message; }
        return out;
      })()`,
      returnByValue: true,
      awaitPromise: true,
    });
    const vc = vocabRt.result.value;
    check('vocab.policy_served',
      vc.status === 200 && vc.total >= 30 && vc.moneyDoor >= 8 && vc.checkpoint >= 6 && vc.auditLog >= 20 && vc.rulesOk === true,
      'status=' + vc.status + ' total=' + vc.total + ' moneyDoor=' + vc.moneyDoor + ' checkpoint=' + vc.checkpoint +
      ' auditLog=' + vc.auditLog + ' rules=' + vc.rulesOk + ' err=' + (vc.error || ''));
    check('vocab.post_is_refused', vc.postStatus === 405, 'postStatus=' + vc.postStatus);
    check('vocab.prefixes_are_exact_and_unambiguous',
      Array.isArray(vc.ambiguous) && vc.ambiguous.length === 0,
      'ambiguous=' + JSON.stringify(vc.ambiguous || []));
    check('vocab.client_reads_the_served_vocabulary',
      vc.hasModule === true && vc.ready === true && typeof vc.built === 'string' && vc.built.indexOf(':') > 0 &&
      vc.doorCount === vc.moneyDoor,
      'module=' + vc.hasModule + ' ready=' + vc.ready + ' built=' + vc.built + ' doors=' + vc.doorCount + '/' + vc.moneyDoor);
    check('vocab.client_fails_closed_on_a_bad_purpose',
      vc.unknownKeyRefused === true && vc.emptyPartRefused === true,
      'unknownKey=' + vc.unknownKeyRefused + ' emptyPart=' + vc.emptyPartRefused);
    check('vocab.client_declares_no_prefix',
      vc.jsStatus === 200 && Array.isArray(vc.spelledPrefixes) && vc.spelledPrefixes.length === 0,
      'jsStatus=' + vc.jsStatus + ' spelled=' + JSON.stringify(vc.spelledPrefixes || []));



    // --- THE CAREER PATH: the served taxonomy, the one missing gate, and a client that re-declares none of it ------
    // WHY THIS IS NOT A DUPLICATE OF THE GO SUITE. `career_path_test.go` pins the DOMAIN and the HTTP boundary
    // (401/400/403/405, the warn/grace/demote lifecycle, the agreement between the path map and the rival-pair
    // matrix). What neither the Go suite nor the handler gate can see is the tier this probe exists for:
    //   (a) does the served taxonomy actually REACH a rendered panel (the fabricated-taxonomy class: this module
    //       carried a `faction: JUSTICE|UNDERWORLD|HYBRID` list the server had never served),
    //   (b) is the World Dashboard leaf's opener really PUBLISHED (the dead-leaf class), and
    //   (c) does the module still declare a taxonomy of its own in its CODE.
    // So every assertion below compares what is RENDERED against what was SERVED in the same run - never against a
    // list typed into this file, except the vocabulary facts the operator specified (justice/criminal/neutral,
    // user/manager/governor, ONE direct rivalry), which are the CONTRACT and must fail loudly if they drift.
    const careerRt = await cdp.send('Runtime.evaluate', {
      expression: `(async () => {
        const out = {};
        // A UNIQUE wallet per run: the record is guaranteed virgin, so no prior promotion can make a gate vanish
        // (and a wallet with no quota keys the same limiter bucket either way, so this costs nothing).
        const W = 'probe-career-' + Date.now();
        const prevWallet = window.currentWallet;
        const sleep = (ms) => new Promise(function (r) { setTimeout(r, ms); });
        const jfetch = async function (url, opts) {
          let r = await fetch(url, opts);
          for (let i = 0; i < 3 && r.status === 429; i++) { await sleep(1500); r = await fetch(url, opts); }
          return r;
        };
        const jsonOf = async function (r) { try { return await r.json(); } catch (e) { return null; } };
        try {
          // 1. THE THRESHOLD. All three doors resolve the caller the way every other door does, so an anonymous
          // request must be refused BY THE DOOR instead of being answered with a taxonomy.
          const anonGet = await fetch('/api/career/path');
          out.anonGet = anonGet.status;
          const anonChoose = await fetch('/api/career/path/choose', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ path: 'justice' }) });
          out.anonChoose = anonChoose.status;
          const anonPromote = await fetch('/api/career/promote', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ role: 'Judge' }) });
          out.anonPromote = anonPromote.status;
          // 2. THE CLIENT MAY NAME A PATH OR A ROLE AND NOTHING ELSE. The decoder runs BEFORE any lookup, so a body
          // that tries to hand itself a civil rank is refused even with no wallet in play.
          const owned = await fetch('/api/career/path/choose', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ path: 'justice', civil_tier: 'governor' }) });
          out.ownedField = owned.status;
          out.ownedError = String(((await jsonOf(owned)) || {}).error || '');
          // 3. THE READ (the served projection the panel renders).
          const resp = await jfetch('/api/career/path?wallet=' + W);
          out.status = resp.status;
          const body = await jsonOf(resp);
          const d = (body && body.data) || {};
          out.hasData = !!body && body.success === true && !!body.data;
          const paths = Array.isArray(d.paths) ? d.paths : [];
          out.pathIds = paths.map(function (p) { return p.id; });
          out.pathLabels = paths.map(function (p) { return p.label; });
          out.pathsSummarised = paths.length > 0 && paths.every(function (p) { return String(p.summary || '').length > 0; });
          const cells = [];
          paths.forEach(function (p) {
            (p.rivals || []).forEach(function (r) { cells.push({ from: p.id, to: r.against, rel: r.relation, bps: r.bonus_bps, explain: r.explain }); });
          });
          out.cells = cells.length;
          out.vocab = Array.from(new Set(cells.map(function (c) { return c.rel; }))).sort();
          out.direct = cells.filter(function (c) { return c.rel === 'direct'; }).map(function (c) { return c.from + '->' + c.to; }).sort();
          out.directBps = Array.from(new Set(cells.filter(function (c) { return c.rel === 'direct'; }).map(function (c) { return c.bps; }))).sort();
          out.otherBps = Array.from(new Set(cells.filter(function (c) { return c.rel !== 'direct'; }).map(function (c) { return c.bps; })));
          out.neutralRels = Array.from(new Set(cells.filter(function (c) { return c.from === 'neutral'; }).map(function (c) { return c.rel; }))).sort();
          out.cellExplained = cells.length > 0 && cells.every(function (c) { return String(c.explain || '').length > 0; });
          out.civilTiers = (d.civil_ranks || []).map(function (r) { return r.tier; });
          out.civilRequires = (d.civil_ranks || []).length === 3 && (d.civil_ranks || []).every(function (r) { return String(r.requires || '').length > 0; });
          out.civilNow = d.civil ? String(d.civil.tier) : '';
          // A wallet that owns no club, no territory and no region IS a user. The rank is DERIVED, never declared.
          out.civilDerived = out.civilNow === out.civilTiers[0];
          const careers = Array.isArray(d.careers) ? d.careers : [];
          out.careers = careers.length;
          out.ineligible = careers.filter(function (c) { return c.eligible !== true && c.promoted !== true; }).length;
          // An ineligible career must SAY why: a blank reason renders as "locked" with nothing to act on.
          out.everyIneligibleSaysWhy = careers.every(function (c) { return c.eligible === true || c.promoted === true || String(c.missing || '').length > 0; });
          out.anyMissingReason = careers.some(function (c) { return String(c.missing || '').length > 0; });
          out.homePaths = Array.from(new Set(careers.map(function (c) { return c.home_path; }))).sort();
          const mx = Array.isArray(d.matrices) ? d.matrices : [];
          out.matrixStages = mx.map(function (m) { return m.stage; });
          out.matrixOwners = mx.map(function (m) { return String(m.owner || ''); });
          out.matrixOwnersOk = mx.length > 0 && mx.every(function (m) { return String(m.owner || '').length > 0 && String(m.describes || '').length > 0; });
          out.pairs = (d.career_pairs || []).length;
          out.pairAPaths = Array.from(new Set((d.career_pairs || []).map(function (p) { return p.a; }))).sort();
          out.rules = Object.keys(d.rules || {});
          // 4. THE CLIENT RENDERS WHAT WAS SERVED. The overlay is opened the way the World Dashboard leaf opens it,
          // and its TEXT is compared against the payload above. A drained bucket is retried with a real wait: a
          // limiter reading is not a product failure, and the client's own refusal is DISTINGUISHED below, never
          // silently read as an empty panel.
          window.currentWallet = W;
          let rendered = 0, attempts = 0;
          out.published = typeof window.openCareers === 'function';
          while (attempts < 6 && !rendered) {
            attempts++;
            if (out.published) window.openCareers();
            await sleep(1600);
            const c0 = document.getElementById('careers-content');
            const t0 = c0 ? (c0.textContent || '') : '';
            const seen = out.pathLabels.filter(function (l) { return l && t0.indexOf(l) !== -1; }).length;
            if (out.pathLabels.length > 0 && seen === out.pathLabels.length) rendered = 1;
          }
          out.renderAttempts = attempts;
          const ov = document.getElementById('careers-overlay');
          out.overlayDisplay = ov ? getComputedStyle(ov).display : 'absent';
          const content = document.getElementById('careers-content');
          const text = content ? (content.textContent || '') : '';
          out.labelsPainted = out.pathLabels.filter(function (l) { return l && text.indexOf(l) !== -1; }).length;
          out.rowsPainted = document.querySelectorAll('#careers-content .career-row').length;
          out.missingPainted = document.querySelectorAll('#careers-content .career-missing').length;
          out.promotePainted = document.querySelectorAll('#careers-content .career-pick-role').length;
          out.civilPainted = document.querySelectorAll('#careers-content .career-civil-rank').length;
          out.matrixOwnersPainted = document.querySelectorAll('#careers-content .career-matrix-owner').length;
          out.ruleRowsPainted = document.querySelectorAll('#careers-content .career-rule').length;
          out.heldPainted = document.querySelectorAll('#careers-content .career-held-badge').length;
          out.actionsPublished = typeof window.chooseCareerPath === 'function' && typeof window.promoteCareerRole === 'function';
          out.refusalStated = text.indexOf('could not be read') !== -1;
          // 5. THE LEAF. The World Dashboard must declare the feature, route OUT of the dashboard, and name an
          // opener that is really published. WDTaxonomy is published at import; the wait below keeps this an
          // assertion about the LEAF rather than about the probe's timing.
          let tax = window.WDTaxonomy;
          for (let i = 0; i < 10 && !tax; i++) { await sleep(300); tax = window.WDTaxonomy; }
          out.wdFeature = false; out.wdOpener = null; out.wdRoutesOut = false;
          if (tax) {
            for (const cat of tax.categories) {
              for (const f of cat.features) {
                if (f.tab === 'career') { out.wdFeature = true; out.wdOpener = f.opener; out.wdRoutesOut = f.routes_out === true; }
              }
            }
          }
          out.wdOpenerPublished = out.wdOpener ? typeof window[out.wdOpener] === 'function' : false;
          // 6. THE CLIENT DECLARES NO TAXONOMY. FULL-LINE COMMENTS ARE STRIPPED FIRST: this module now DOCUMENTS
          // the taxonomy it removed in its header, and a comment must never read as a declaration.
          const jsr = await fetch('/js/career_tree.js');
          out.jsStatus = jsr.status;
          const src = jsr.ok ? await jsr.text() : '';
          const code = src.split(/\\r?\\n/).filter(function (l) { return !/^\\s*\\/\\//.test(l); }).join('\\n');
          out.factionTokens = ['JUSTICE', 'UNDERWORLD', 'HYBRID'].filter(function (t) { return code.indexOf(t) !== -1; });
          out.factionField = /faction\\s*:/.test(code);
          // 7. THE UNLOCK ADVISORY — "unlocked, never forced" (career_path.go §7.5). The operator's rule is a
          // RESTRICTION on what the code may do, so it is asserted three ways: what the server SERVES (optional,
          // with its own basis), that the SERVED unlock AGREES with the SERVED career table, and that the panel
          // paints the SERVED words. promotePainted is then compared with the SERVED unlock count, so a control
          // can exist only for a career the level cap has actually opened — the whole of "not mandatory".
          const adv = d.upgrades || {};
          out.advIsOptional = adv.is_optional === true;
          out.advStatement = String(adv.statement || '');
          out.advUnlockBasis = String(adv.unlock_basis || '');
          out.advNoticeRule = String(adv.notice_rule || '');
          out.advStaffBasis = String(adv.staff_basis || '');
          out.advUnlockedCount = adv.unlocked_count;
          out.advUnlockedRoles = Array.isArray(adv.unlocked_roles) ? adv.unlocked_roles.length : -1;
          out.advStaffIsArray = Array.isArray(adv.staff);
          out.advStaffLen = Array.isArray(adv.staff) ? adv.staff.length : -1;
          out.advRequestsIsArray = Array.isArray(adv.requests_to_me);
          out.advCanRequestStaff = adv.can_request_staff === true;
          out.eligibleCareerRoles = careers.filter(function (c) { return c.eligible === true; });
          out.unlockAgreesWithGates = out.advUnlockedRoles === out.eligibleCareerRoles.length;
          out.rulesDeclareOptIn = out.rules.indexOf('upgrade_is_opt_in') >= 0 && out.rules.indexOf('notice_only') >= 0 &&
            out.rules.indexOf('unlock_basis') >= 0 && out.rules.indexOf('staff_upgrades') >= 0;
          out.unlockStatementPainted = document.querySelectorAll('#careers-content .career-unlock-statement').length;
          out.staffBasisPainted = document.querySelectorAll('#careers-content .career-staff-basis').length;
          out.staffRowsPainted = document.querySelectorAll('#careers-content .career-staff-row').length;
          out.staffRequestButtons = document.querySelectorAll('#careers-content .career-staff-request').length;
          out.statementRendered = out.advStatement.length > 0 && text.indexOf(out.advStatement.slice(0, 40)) !== -1;
          out.staffActionPublished = typeof window.requestStaffCareerUpgrade === 'function';
          // 8. THE EMPLOYER'S DOOR. It ASKS and can never grant: the wrong verb is refused, an anonymous caller is
          // refused, a body that names a role grant is refused at the DECODER (before any lookup), and a caller who
          // employs nobody is refused with the reason it can state. Every wallet in this run owns no club, which is
          // what makes that last one a real refusal rather than a fixture.
          const staffBody = JSON.stringify({ staff_wallet: '0xprobe-staff', role: 'Warden' });
          const staffHeaders = { 'Content-Type': 'application/json' };
          out.staffGet = (await jfetch('/api/career/staff/request?wallet=' + W)).status;
          out.staffAnon = (await jfetch('/api/career/staff/request', { method: 'POST', headers: staffHeaders, body: staffBody })).status;
          out.staffOwnedField = (await jfetch('/api/career/staff/request', {
            method: 'POST', headers: staffHeaders,
            body: JSON.stringify({ staff_wallet: '0xprobe-staff', role: 'Warden', promoted_roles: ['Judge'] }),
          })).status;
          const postNoClub = await jfetch('/api/career/staff/request?wallet=' + W, {
            method: 'POST', headers: staffHeaders, body: staffBody,
          });
          out.staffNoClub = postNoClub.status;
          const noClubBody = await jsonOf(postNoClub);
          out.staffNoClubError = String((noClubBody && noClubBody.error) || '');
          out.staffNoClubCarriesAdvisory = !!(noClubBody && noClubBody.upgrades);
          // Leave the probe exactly as it was found: no probe-only wallet, no open overlay.
          window.currentWallet = prevWallet;
          if (typeof window.hideAllOverlays === 'function') window.hideAllOverlays();
        } catch (e) { out.error = String(e && e.message ? e.message : e); }
        return out;
      })()`,
      returnByValue: true,
      awaitPromise: true,
    });
    const cr = careerRt.result.value;
    check('career.doors_refuse_an_anonymous_caller',
      cr.anonGet === 401 && cr.anonChoose === 401 && cr.anonPromote === 401,
      'get=' + cr.anonGet + ' choose=' + cr.anonChoose + ' promote=' + cr.anonPromote);
    check('career.client_cannot_declare_its_own_gate',
      cr.ownedField === 400 && String(cr.ownedError || '').indexOf('invalid body') === 0,
      'status=' + cr.ownedField + ' error=' + cr.ownedError);
    check('career.read_serves_the_whole_taxonomy',
      cr.status === 200 && cr.hasData === true &&
      (cr.pathIds || []).length === 3 && cr.cells === 9 &&
      (cr.civilTiers || []).length === 3 && cr.civilRequires === true && cr.civilDerived === true &&
      cr.careers > 0 && cr.everyIneligibleSaysWhy === true && cr.anyMissingReason === true &&
      (cr.matrixStages || []).length === 3 && cr.matrixOwnersOk === true && cr.pairs > 0 &&
      (cr.rules || []).length > 0 && cr.pathsSummarised === true && cr.cellExplained === true,
      'status=' + cr.status + ' paths=' + JSON.stringify(cr.pathIds || []) + ' cells=' + cr.cells +
      ' ranks=' + JSON.stringify(cr.civilTiers || []) + ' careers=' + cr.careers + ' ineligible=' + cr.ineligible +
      ' matrices=' + JSON.stringify(cr.matrixStages || []) + ' pairs=' + cr.pairs + ' rules=' + (cr.rules || []).length +
      ' homes=' + JSON.stringify(cr.homePaths || []) + ' err=' + (cr.error || ''));
    check('career.three_paths_and_the_one_direct_rivalry',
      JSON.stringify(cr.pathIds) === JSON.stringify(['justice', 'criminal', 'neutral']) &&
      JSON.stringify(cr.direct) === JSON.stringify(['criminal->justice', 'justice->criminal']) &&
      JSON.stringify(cr.neutralRels) === JSON.stringify(['interpreted']) &&
      JSON.stringify(cr.directBps) === JSON.stringify([1000]) &&
      JSON.stringify(cr.otherBps) === JSON.stringify([0]),
      'paths=' + JSON.stringify(cr.pathIds) + ' direct=' + JSON.stringify(cr.direct) + ' neutral=' + JSON.stringify(cr.neutralRels) +
      ' directBps=' + JSON.stringify(cr.directBps) + ' otherBps=' + JSON.stringify(cr.otherBps) + ' vocab=' + JSON.stringify(cr.vocab));
    check('career.civil_ladder_is_user_manager_governor',
      JSON.stringify(cr.civilTiers) === JSON.stringify(['user', 'manager', 'governor']) && cr.civilDerived === true,
      'ranks=' + JSON.stringify(cr.civilTiers) + ' now=' + cr.civilNow + ' derived=' + cr.civilDerived);
    // A number is never served without the LAYER it came from, so each matrix names the file that owns it.
    check('career.the_three_matrices_name_their_owners',
      (cr.matrixOwners || []).length === 3 &&
      String(cr.matrixOwners[0]).indexOf('career_path.go') >= 0 &&
      String(cr.matrixOwners[1]).indexOf('rival_career_engine.go') >= 0 &&
      String(cr.matrixOwners[2]).indexOf('rivalry_engine.go') >= 0,
      'stages=' + JSON.stringify(cr.matrixStages) + ' owners=' + JSON.stringify(cr.matrixOwners || []));
    check('career.client_renders_the_served_taxonomy',
      cr.published === true && cr.overlayDisplay === 'flex' &&
      cr.labelsPainted === (cr.pathLabels || []).length &&
      cr.rowsPainted === cr.careers && cr.missingPainted === cr.ineligible &&
      cr.civilPainted === (cr.civilTiers || []).length &&
      cr.matrixOwnersPainted === (cr.matrixStages || []).length &&
      cr.ruleRowsPainted === (cr.rules || []).length &&
      cr.actionsPublished === true && cr.heldPainted === 0,
      'published=' + cr.published + ' display=' + cr.overlayDisplay + ' labels=' + cr.labelsPainted + '/' + (cr.pathLabels || []).length +
      ' rows=' + cr.rowsPainted + '/' + cr.careers + ' missing=' + cr.missingPainted + '/' + cr.ineligible +
      ' promoteBtns=' + cr.promotePainted + ' civil=' + cr.civilPainted + ' owners=' + cr.matrixOwnersPainted +
      ' rules=' + cr.ruleRowsPainted + '/' + (cr.rules || []).length + ' actions=' + cr.actionsPublished +
      ' attempts=' + cr.renderAttempts + ' refusal=' + cr.refusalStated);
    check('career.the_leaf_names_a_published_opener',
      cr.wdFeature === true && cr.wdRoutesOut === true && cr.wdOpener === 'openCareers' && cr.wdOpenerPublished === true,
      'feature=' + cr.wdFeature + ' routesOut=' + cr.wdRoutesOut + ' opener=' + cr.wdOpener + ' published=' + cr.wdOpenerPublished);
    // The fabricated-taxonomy class, asserted in CODE (a comment may still name what was removed).
    check('career.client_declares_no_taxonomy',
      cr.jsStatus === 200 && (cr.factionTokens || []).length === 0 && cr.factionField === false,
      'jsStatus=' + cr.jsStatus + ' tokens=' + JSON.stringify(cr.factionTokens || []) + ' factionField=' + cr.factionField);
    // "UNLOCKED, NEVER FORCED" — the operator's rule, asserted as a RESTRICTION rather than as a feature: the
    // served advisory is optional and states its own basis, the served unlock AGREES with the served career
    // table, the panel paints the SERVED words, and an upgrade control exists ONLY for a career the level cap
    // has opened (promotePainted is compared with the SERVED unlock count, so a control can never appear for a
    // career the table below it says is locked).
    check('career.upgrade_is_unlocked_not_forced',
      cr.advIsOptional === true && (cr.advStatement || '').length > 0 && (cr.advUnlockBasis || '').length > 0 &&
      (cr.advNoticeRule || '').length > 0 && cr.rulesDeclareOptIn === true && cr.unlockAgreesWithGates === true &&
      cr.unlockStatementPainted === 1 && cr.statementRendered === true &&
      cr.promotePainted === cr.advUnlockedCount,
      'optional=' + cr.advIsOptional + ' unlocked=' + cr.advUnlockedCount + '/' + (cr.eligibleCareerRoles || []).length +
      ' promoteBtns=' + cr.promotePainted + ' statementPainted=' + cr.unlockStatementPainted +
      ' rendered=' + cr.statementRendered + ' rules=' + cr.rulesDeclareOptIn +
      ' unlockBasis="' + String(cr.advUnlockBasis || '').slice(0, 60) + '"');
    // THE STAFF PROJECTION and the EMPLOYER'S DOOR. The projection always names its BASIS (so "staff" is never a
    // guess), and no wallet in this run owns a club, so the served truth is "you employ nobody" — rendered, not
    // an empty box. The door ASKS: it can never grant a role, so it refuses the wrong verb, an anonymous caller,
    // a body that names a role grant, and a caller who employs nobody.
    check('career.staff_projection_names_its_basis',
      cr.advStaffIsArray === true && cr.advStaffLen === 0 && cr.advRequestsIsArray === true &&
      cr.advCanRequestStaff === false && (cr.advStaffBasis || '').length > 0 &&
      cr.staffBasisPainted === 1 && cr.staffRowsPainted === cr.advStaffLen &&
      cr.staffRequestButtons === 0 && cr.staffActionPublished === true,
      'staff=' + cr.advStaffLen + ' canRequest=' + cr.advCanRequestStaff + ' basisPainted=' + cr.staffBasisPainted +
      ' rows=' + cr.staffRowsPainted + ' requestBtns=' + cr.staffRequestButtons + ' published=' + cr.staffActionPublished);
    check('career.staff_request_asks_and_cannot_grant',
      cr.staffGet === 405 && cr.staffAnon === 401 && cr.staffOwnedField === 400 &&
      cr.staffNoClub === 403 && String(cr.staffNoClubError || '').indexOf('own no club') >= 0 &&
      cr.staffNoClubCarriesAdvisory === true,
      'get=' + cr.staffGet + ' anon=' + cr.staffAnon + ' ownedField=' + cr.staffOwnedField +
      ' noClub=' + cr.staffNoClub + ' err="' + cr.staffNoClubError + '" advisory=' + cr.staffNoClubCarriesAdvisory);




    // --- Single-Navigation + TWO-TIER Navigation Mandate ---------------------
    // 1. TIER 1 shows CATEGORIES ONLY — no leaf/feature button is rendered at all.
    // 2. Choosing a category reveals that category's TIER 2 feature menu.
    // 3. Stars live on CATEGORIES (quick access = categories, not the leaves).
    // 4. The Player Profile is reachable FROM the dashboard (WD owns access).
    const wdStep = await cdp.send('Runtime.evaluate', {
      expression: `(() => {
        if (typeof window.openWorldDashboard === 'function') window.openWorldDashboard();
        const cats = Array.from(document.querySelectorAll('.wd-cat-btn'));
        const leavesAtTier1 = document.querySelectorAll('.wd-tab').length;
        const starCount = document.querySelectorAll('.wd-cat-star').length;
        const playerHub = cats.filter(b => (b.textContent || '').indexOf('Player Hub') >= 0)[0];
        if (playerHub) playerHub.click();                       // drill into TIER 2
        const featureTabs = Array.from(document.querySelectorAll('.wd-tab')).map(b => b.dataset.tab);
        const menu = document.getElementById('wd-feature-menu');
        return {
          wdHasPortfolioTab: featureTabs.indexOf('portfolio') >= 0,
          categoryCount: cats.length,
          leavesAtTier1: leavesAtTier1,
          starCount: starCount,
          tier2FeatureCount: featureTabs.length,
          tier2MenuVisible: !!menu && !menu.classList.contains('hidden'),
        };
      })()`,
      returnByValue: true,
    });
    await new Promise((r) => setTimeout(r, 1200));

    await cdp.send('Runtime.evaluate', {
      expression: `(() => {
        if (typeof window.openPortfolio === 'function') window.openPortfolio('overview');
        return true;
      })()`,
      returnByValue: true,
    });
    await new Promise((r) => setTimeout(r, 2500));

    const ro = await cdp.send('Runtime.evaluate', {
      expression: `(() => ({
        expandButtons: document.querySelectorAll('.pp-expand-btn').length,
        rmButtons: document.querySelectorAll('.pp-rm-btn').length,
        readOnlyHints: Array.from(document.querySelectorAll('.pp-rm-hint')).filter(e => /World Dashboard/.test(e.textContent || '')).length,
        diag: {
          hubPresent: !!document.querySelector('.pp-hub'),
          hubDisplay: (() => { const el = document.querySelector('.pp-hub'); return el ? getComputedStyle(el).display : 'absent'; })(),
          activeCat: (document.querySelector('.pp-cat.active') || { dataset: {} }).dataset ? (document.querySelector('.pp-cat.active') || {}).dataset?.cat : null,
          activeSub: (() => { const b = document.querySelector('#pp-subtabs .pp-subtab.active'); return b ? b.dataset.sub : null; })(),
          subtabs: Array.from(document.querySelectorAll('#pp-subtabs .pp-subtab')).map(b => b.dataset.sub),
          hintAny: document.querySelectorAll('.pp-rm-hint').length,
          panelText: (() => { const p = document.querySelector('#pp-panel'); return p ? (p.textContent || '').slice(0, 160) : 'no-pp-panel'; })(),
        },
      }))()`,
      returnByValue: true,
    });

    check('wd.tier1_renders_categories', wdStep.result.value.categoryCount >= 8, wdStep.result.value.categoryCount);
    check('wd.tier1_hides_leaf_features', wdStep.result.value.leavesAtTier1 === 0, wdStep.result.value.leavesAtTier1);
    check('wd.stars_on_categories_only', wdStep.result.value.starCount === wdStep.result.value.categoryCount, wdStep.result.value.starCount + '/' + wdStep.result.value.categoryCount);
    check('wd.tier2_feature_menu_opens', wdStep.result.value.tier2MenuVisible === true && wdStep.result.value.tier2FeatureCount > 0, wdStep.result.value.tier2FeatureCount);
    check('wd.owns_portfolio_route_tab', wdStep.result.value.wdHasPortfolioTab === true, wdStep.result.value.wdHasPortfolioTab);
    check('portfolio.no_territory_purchase_control', ro.result.value.expandButtons === 0, ro.result.value.expandButtons);
    check('portfolio.no_rm_activate_control', ro.result.value.rmButtons === 0, ro.result.value.rmButtons);
    check('portfolio.read_only_hint_points_to_wd', ro.result.value.readOnlyHints > 0, ro.result.value.readOnlyHints);

    // --- Portfolio analytics contract (READ-ONLY PORTFOLIO MANDATE) ----------
    // The Portfolio is an ANALYTICS surface: every category must expose at least
    // one screen, and every screen must render real content - never an empty
    // panel and never a render error. It must expose no transacting controls.
    const pf = await cdp.send('Runtime.evaluate', {
      expression: `(async () => {
        const cats = Array.from(document.querySelectorAll('.pp-cat'));
        const results = [];
        for (const c of cats) {
          c.click();
          await new Promise(r => setTimeout(r, 260));
          const subs = Array.from(document.querySelectorAll('#pp-subtabs .pp-subtab'));
          const panel = document.getElementById('pp-panel');
          const text = panel ? (panel.textContent || '') : '';
          results.push({ cat: c.dataset.cat, screens: subs.length, chars: text.trim().length, errored: /Failed to render/i.test(text) });
        }
        const actionControls = document.querySelectorAll('.pp-action-btn, .pp-expand-btn, .pp-rm-btn').length;
        const hub = document.querySelector('.pp-hub');
        const cs = hub ? getComputedStyle(hub) : null;
        return {
          categoryCount: cats.length,
          minScreens: results.reduce((m, r) => Math.min(m, r.screens), 99),
          emptyScreens: results.filter(r => r.chars === 0).map(r => r.cat),
          errored: results.filter(r => r.errored).map(r => r.cat),
          actionControls: actionControls,
          hubDisplay: cs ? cs.display : 'absent',
          hubVisibility: cs ? cs.visibility : 'absent',
          hubOpacity: cs ? cs.opacity : 'absent'
        };
      })()`,
      returnByValue: true,
      awaitPromise: true,
    });
    const pv = pf.result.value;
    check('portfolio.analytics_categories_rendered', pv.categoryCount >= 10, pv.categoryCount);
    check('portfolio.every_category_has_screens', pv.minScreens >= 1, 'minScreens=' + pv.minScreens);
    check('portfolio.no_empty_screens', pv.emptyScreens.length === 0, JSON.stringify(pv.emptyScreens));
    check('portfolio.no_render_errors', pv.errored.length === 0, JSON.stringify(pv.errored));
    check('portfolio.is_read_only_no_action_controls', pv.actionControls === 0, pv.actionControls);
    // Regression guard: the hub must actually be VISIBLE. `hideAllOverlays()` adds
    // `.hidden` (`display:none !important`) to every `.overlay`, which silently made
    // this surface un-openable until the class was cleared on show.
    check('portfolio.hub_is_visible',
      pv.hubDisplay !== 'none' && pv.hubVisibility !== 'hidden' && String(pv.hubOpacity) !== '0',
      pv.hubDisplay + ' / vis:' + pv.hubVisibility + ' / op:' + pv.hubOpacity);
    // --- Portfolio: EVERY screen of EVERY category must render --------------
    // The category sweep above only renders each category's DEFAULT screen. The
    // associations mandate requires every analytical screen (rivalry, companions,
    // clubs/alliances, bonded assets, faith, custody, obligations, …) to render
    // real content too — never an empty panel and never a render error.
    const pfAll = await cdp.send('Runtime.evaluate', {
      expression: `(async () => {
        const empty = [];
        const errored = [];
        let screens = 0;
        const cats = Array.from(document.querySelectorAll('.pp-cat'));
        for (const c of cats) {
          c.click();
          await new Promise(r => setTimeout(r, 70));
          // Screen 0 is already covered by the category sweep above.
          const subs = Array.from(document.querySelectorAll('#pp-subtabs .pp-subtab')).slice(1);
          for (const s of subs) {
            s.click();
            await new Promise(r => setTimeout(r, 110));
            screens++;
            const panel = document.getElementById('pp-panel');
            const text = panel ? (panel.textContent || '') : '';
            if (text.trim().length === 0) empty.push(c.dataset.cat + '/' + s.dataset.sub);
            if (/Failed to render/i.test(text)) errored.push(c.dataset.cat + '/' + s.dataset.sub);
          }
        }
        return { screens: screens, empty: empty, errored: errored };
      })()`,
      returnByValue: true,
      awaitPromise: true,
    });
    const pfa = pfAll.result.value;
    check('portfolio.every_screen_renders', pfa.empty.length === 0, 'screens=' + pfa.screens + ' empty=' + JSON.stringify(pfa.empty));
    check('portfolio.no_screen_render_errors', pfa.errored.length === 0, JSON.stringify(pfa.errored));

    // --- Lazy panel materialisation (regression guard) -----------------------
    // These features used to dead-end on a "Not Wired Yet" placeholder purely
    // because no `wd-panel-<tab>` container existed, even though their initXxx()
    // module was fully built and renders into `#wd-<tab>`. Each must now
    // materialise a real panel instead of a placeholder.
    const lazy = await cdp.send('Runtime.evaluate', {
      expression: `(() => {
        const tabs = ['game','create_match','deck','campaign','locations','tutorial',
                      'card_titles','wagers','mood','tea_house','zen_garden','npc_taunts',
                      'game_modes','multiplayer','equipment','card_progression'];
        const missingPanel = [], emptyPanel = [];
        tabs.forEach(t => {
          if (typeof window.switchWDTab === 'function') window.switchWDTab(t);
          const p = document.getElementById('wd-panel-' + t);
          const inner = document.getElementById('wd-' + t);
          if (!p || !inner) missingPanel.push(t);
          else if (!inner.innerHTML || inner.innerHTML.length === 0) emptyPanel.push(t);
        });
        return { total: tabs.length, missingPanel, emptyPanel };
      })()`,
      returnByValue: true,
    });
    check('wd.lazy_panels_materialise', lazy.result.value.missingPanel.length === 0, JSON.stringify(lazy.result.value.missingPanel));
    // 15/16 render immediately; `multiplayer` renders only once match state exists.
    check('wd.recovered_features_render', lazy.result.value.emptyPanel.length <= 1, 'empty=' + JSON.stringify(lazy.result.value.emptyPanel));
    console.log('  DIAG  recovered ' + (lazy.result.value.total - lazy.result.value.emptyPanel.length) + '/' + lazy.result.value.total + ' feature tabs render into a real panel');

    // --- Quick access round trip: category star -> favorites -----------------
    // Starring is offered on CATEGORIES only, and every stored star must resolve
    // to a category (never to a leaf feature).
    const starRt = await cdp.send('Runtime.evaluate', {
      expression: `(() => {
        const star = document.querySelector('.wd-cat-star');
        if (!star) return { ok: false, reason: 'no category star rendered' };
        const catId = star.dataset.starCat;
        star.click();                                     // star the category
        const stored = !!(window.UserPreferences && window.UserPreferences.isStarred(catId, 'main'));
        const stars = (window.UserPreferences && window.UserPreferences.getStarred) ? window.UserPreferences.getStarred() : [];
        const allStarsAreCategories = stars.every(s => !!document.querySelector('.wd-cat-btn[data-cat="' + s.wdTab + '"]'));
        star.click();                                     // unstar -> leave state clean
        return { ok: true, catId, stored, allStarsAreCategories, starCount: stars.length };
      })()`,
      returnByValue: true,
    });
    check('quick_access.starring_a_category_persists', starRt.result.value.ok === true && starRt.result.value.stored === true, JSON.stringify(starRt.result.value));
    check('quick_access.no_leaf_feature_starred', starRt.result.value.allStarsAreCategories === true, starRt.result.value.allStarsAreCategories);
    // --- Taxonomy coverage: every built surface has a home --------------------
    // AI Citizens and Children Bots were fully built but had NO World Dashboard
    // home: `window.openAICitizens` was published and called from nowhere, and
    // Children Bots were reachable only through the constellation hub. Both now
    // live under Assets — the owned-and-developed domain, alongside the Kennel and
    // Garage — and the ↗ badge must say they route out.
    const taxonomy = await cdp.send('Runtime.evaluate', {
      expression: `(() => {
        const cat = document.querySelector('.wd-cat-btn[data-cat="assets"]');
        if (cat) cat.click();
        const btns = Array.from(document.querySelectorAll('.wd-feature-btn'));
        const tabs = btns.map(b => b.dataset.tab);
        const badgeOf = (t) => {
          const b = btns.find(x => x.dataset.tab === t);
          const badge = b ? b.querySelector('.wd-feature-badge') : null;
          return badge ? (badge.textContent || '').trim() : null;
        };
        return {
          assetsTabCount: btns.filter(b => b.dataset.cat === 'assets').length,
          hasCitizens: tabs.indexOf('citizens') >= 0,
          hasChildren: tabs.indexOf('children') >= 0,
          citizensBadge: badgeOf('citizens'),
          childrenBadge: badgeOf('children'),
          openersPublished: {
            citizens: typeof window.openAICitizens === 'function',
            children: typeof window.openChildrenBots === 'function',
          },
          governanceOpenerPublished: typeof window.openGovernancePanel === 'function',
        };
      })()`,
      returnByValue: true,
    });
    check('wd.citizens_and_children_have_a_home',
      taxonomy.result.value.hasCitizens === true && taxonomy.result.value.hasChildren === true,
      JSON.stringify({ citizens: taxonomy.result.value.hasCitizens, children: taxonomy.result.value.hasChildren }));
    check('wd.those_features_route_out_to_a_published_overlay',
      taxonomy.result.value.openersPublished.citizens === true && taxonomy.result.value.openersPublished.children === true,
      JSON.stringify(taxonomy.result.value.openersPublished));
    check('wd.routed_features_show_the_route_badge',
      taxonomy.result.value.citizensBadge === '↗' && taxonomy.result.value.childrenBadge === '↗',
      JSON.stringify([taxonomy.result.value.citizensBadge, taxonomy.result.value.childrenBadge]));
    check('hub.panel_opener_names_resolve',
      taxonomy.result.value.governanceOpenerPublished === true,
      'openGovernancePanel published=' + taxonomy.result.value.governanceOpenerPublished);



    // --- §25.6.1 / §26.4.2 bonded-asset surfaces (Kennel, Garage, 3D arenas) ---------
    // One owner per feature: the KENNEL owns companion purchase/breeding/grooming, the GARAGE
    // owns vehicle purchase/build, and both ARENAS are honest 3D-world placeholders — the
    // fabricated client-side battle simulation they used to run must not come back.
    const bonded = await cdp.send('Runtime.evaluate', {
      awaitPromise: true,
      expression: `(async () => {
        const sleep = ms => new Promise(r => setTimeout(r, ms));
        const out = { kennelPurchase: false, kennelBreed: false, kennelGroom: false,
                      garageClasses: 0, arenaPlaceholder: false, combatSim: null,
                      kennelHtmlLen: 0, kennelHead: '' };
        if (typeof window.switchWDTab === 'function') window.switchWDTab('pets');
        await sleep(1200);   // the Kennel reads /api/pets before it can render its controls
        const kennel = document.getElementById('wd-pets');
        const kh = kennel ? kennel.innerHTML : '';
        out.kennelHtmlLen = kh.length;
        out.kennelHead = kh.replace(/\\s+/g, ' ').slice(0, 140);
        out.kennelPurchase = kh.indexOf('kennelBuy()') !== -1;
        out.kennelBreed = kh.indexOf('kennelBreed()') !== -1;
        out.kennelGroom = typeof window.kennelGroom === 'function';
        if (typeof window.openLifeAssets === 'function') window.openLifeAssets('vehicles');
        await sleep(1200);   // the class table is server-served, so it arrives with the fetch
        const sel = document.getElementById('la-veh-kind');
        out.garageClasses = sel ? sel.options.length : 0;
        const life = document.getElementById('life-assets-overlay');
        out.garageStatus = (document.getElementById('la-status') || {}).textContent || '';
        out.garageHtmlLen = life ? life.innerHTML.length : 0;
        out.garageSelectHtml = sel ? sel.outerHTML.slice(0, 120) : 'no select';
        if (life) life.style.display = 'none';
        if (typeof window.openVehicleArena === 'function') window.openVehicleArena();
        const title = document.getElementById('bonded-arena-title');
        out.arenaPlaceholder = !!(title && title.textContent.indexOf('3D world') !== -1);
        out.combatSim = typeof window.pbaAttack === 'function';
        const arena = document.getElementById('bonded-arena-overlay');
        if (arena) arena.style.display = 'none';
        if (typeof window.hideAllOverlays === 'function') window.hideAllOverlays();
        return out;
      })()`,
      returnByValue: true,
    });
    check('bonded.kennel_owns_purchase', bonded.result.value.kennelPurchase === true, bonded.result.value.kennelPurchase + ' htmlLen=' + bonded.result.value.kennelHtmlLen + ' head=' + bonded.result.value.kennelHead);
    check('bonded.kennel_owns_breeding', bonded.result.value.kennelBreed === true, bonded.result.value.kennelBreed);
    check('bonded.kennel_owns_grooming', bonded.result.value.kennelGroom === true, bonded.result.value.kennelGroom);
    check('bonded.garage_serves_class_table', bonded.result.value.garageClasses >= 3, 'classes=' + bonded.result.value.garageClasses + ' status=' + bonded.result.value.garageStatus + ' select=' + bonded.result.value.garageSelectHtml + ' htmlLen=' + bonded.result.value.garageHtmlLen);
    check('bonded.arena_is_3d_placeholder', bonded.result.value.arenaPlaceholder === true, bonded.result.value.arenaPlaceholder);
    check('bonded.arena_no_combat_simulation', bonded.result.value.combatSim === false, 'pbaAttack=' + bonded.result.value.combatSim);

    // --- §23.5 BONDED BRANDING: ecosphere-wide theming, CARDS EXCLUDED -------------------
    // The SERVER owns the cardinal rule (it refuses card kinds and serves the blocked list), so
    // this probe reads the policy from the API and asks the API to bind a CARD — the client must
    // never be the thing that enforces it, and a card bind must be refused end to end.
    const brandingRt = await cdp.send('Runtime.evaluate', {
      awaitPromise: true,
      expression: `(async () => {
        const sleep = ms => new Promise(r => setTimeout(r, ms));
        const out = { openable: false, display: '', ruleLen: 0, kindButtons: 0, mintForm: false,
                      wdExposesFeature: false, policyKinds: 0, blockedKinds: 0, cardsExcluded: false,
                      cardBindRefused: false, cardError: '',
                      // §10.8 the market the studio renders + the served trade rules.
                      marketFeeBps: 0, marketMin: 0, marketMax: 0, marketRuleServed: false,
                      marketAcquisitionExempt: false, marketListingsArray: false,
                      marketBuyRefusesAPrice: false, marketBuyError: '',
                      marketSellPicker: false, marketPriceInput: false, marketRulePainted: 0,
                      marketListingArea: false };
        out.openable = typeof window.openBondedBranding === 'function';
        if (typeof window.openWorldDashboard === 'function') window.openWorldDashboard();
        if (typeof window.openWDCategory === 'function') window.openWDCategory('assets');
        await sleep(500);
        out.wdExposesFeature = !!document.querySelector('.wd-feature-btn[data-tab="branding"]');
        if (out.openable) await window.openBondedBranding();
        await sleep(1000);
        const ov = document.getElementById('bonded-branding-overlay');
        if (ov) {
          out.display = (getComputedStyle(ov).display || '') + '/' + (ov.style.display || '');
          const rule = document.getElementById('bb-rule');
          out.ruleLen = rule ? rule.textContent.length : 0;
          out.kindButtons = document.querySelectorAll('.bb-kind-btn').length;
          out.mintForm = !!document.getElementById('bb-name');
          ov.style.display = 'none';
        }
        const pol = await (await fetch('/api/assets/targets')).json();
        out.policyKinds = ((pol.policy || {}).bindable_target_kinds || []).length;
        out.blockedKinds = ((pol.policy || {}).blocked_target_kinds || []).length;
        out.cardsExcluded = (pol.policy || {}).cards_excluded === true;
        const tryBody = JSON.stringify({ asset_id: 'BA-x', target_kind: 'deck_card', target_id: '1' });
        const refused = await (await fetch('/api/assets/bind?wallet=probe-branding-wallet', {
          method: 'POST', headers: { 'Content-Type': 'application/json' }, body: tryBody })).json();
        out.cardError = String(refused.error || '');
        out.cardBindRefused = refused.success === false && out.cardError.indexOf('cards are excluded') === 0;
        // ── §10.8 the player-to-player market: the SERVED rules and the studio that renders them ──
        // The trade arithmetic (exact split, sink routing, §27.8 re-proof, write-through) is pinned
        // by the Go tests; this asserts what only a browser can — that the market is reachable, that
        // the studio paints the SERVER's rules, and that a body naming a price is refused.
        out.marketSellPicker = !!document.getElementById('bb-market-asset');
        out.marketPriceInput = !!document.getElementById('bb-market-price');
        // The limiter refills 1 token/sec and this block shares the bucket, so each read retries a
        // 429 instead of reporting a limiter reading as a product failure.
        const jf = async (url, opts) => {
          let r = await fetch(url, opts);
          for (let i = 0; i < 3 && r.status === 429; i++) { await sleep(1500); r = await fetch(url, opts); }
          return r;
        };
        try {
          const vm = await (await jf('/api/assets/market')).json();
          out.marketFeeBps = Number(vm.fee_bps || 0);
          out.marketMin = Number(vm.min_price_micro || 0);
          out.marketMax = Number(vm.max_price_micro || 0);
          out.marketRuleServed = String(vm.rule || '').length > 0;
          out.marketAcquisitionExempt = /ACQUISITION/.test(String(vm.acquisition_rule || ''));
          out.marketListingsArray = Array.isArray(vm.listings);
          const badBuy = await (await jf('/api/assets/market/buy?wallet=probe-market-wallet', {
            method: 'POST', headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ listing_id: 'BML-nope', price_micro: 1 }) })).json();
          out.marketBuyError = String(badBuy.error || '');
          out.marketBuyRefusesAPrice = badBuy.success === false && /only listing_id/i.test(out.marketBuyError);
        } catch (e) { out.marketBuyError = e.message; }
        // The studio's section must render the rules it was served (never its own wording).
        await sleep(2500);
        out.marketRulePainted = ((document.getElementById('bb-market-rule') || {}).textContent || '').length;
        out.marketListingArea = !!document.getElementById('bb-market-listings')
          && !!document.getElementById('bb-market-status');
        if (typeof window.hideAllOverlays === 'function') window.hideAllOverlays();
        return out;
      })()`,
      returnByValue: true,
    });
    check('bonded.branding_studio_opens', brandingRt.result.value.openable === true && brandingRt.result.value.display.indexOf('flex') === 0, 'display=' + brandingRt.result.value.display + ' ruleLen=' + brandingRt.result.value.ruleLen);
    check('bonded.branding_renders_served_policy', brandingRt.result.value.policyKinds >= 9 && brandingRt.result.value.blockedKinds >= 12 && brandingRt.result.value.cardsExcluded === true && brandingRt.result.value.ruleLen > 0, 'kinds=' + brandingRt.result.value.policyKinds + ' blocked=' + brandingRt.result.value.blockedKinds + ' cardsExcluded=' + brandingRt.result.value.cardsExcluded);
    check('bonded.branding_has_mint_form', brandingRt.result.value.mintForm === true && brandingRt.result.value.kindButtons >= 9, 'mintForm=' + brandingRt.result.value.mintForm + ' kindButtons=' + brandingRt.result.value.kindButtons);
    check('bonded.branding_reached_from_the_dashboard', brandingRt.result.value.wdExposesFeature === true, 'feature=' + brandingRt.result.value.wdExposesFeature);
    check('bonded.branding_refuses_card_targets', brandingRt.result.value.cardBindRefused === true, brandingRt.result.value.cardError);
    // §10.8: the market is reachable from the studio, its rules are the SERVER's, and a buy body that
    // tries to name a price is refused by the decoder (a client can never set what it pays).
    check('bonded.market_is_served_and_rendered',
      brandingRt.result.value.marketFeeBps > 0 && brandingRt.result.value.marketMin > 0 && brandingRt.result.value.marketMax > brandingRt.result.value.marketMin &&
      brandingRt.result.value.marketRuleServed === true && brandingRt.result.value.marketAcquisitionExempt === true &&
      brandingRt.result.value.marketListingsArray === true && brandingRt.result.value.marketSellPicker === true &&
      brandingRt.result.value.marketPriceInput === true && brandingRt.result.value.marketRulePainted > 0 &&
      brandingRt.result.value.marketListingArea === true,
      'fee=' + brandingRt.result.value.marketFeeBps + ' min=' + brandingRt.result.value.marketMin + ' max=' + brandingRt.result.value.marketMax +
      ' rulePainted=' + brandingRt.result.value.marketRulePainted + ' err=' + brandingRt.result.value.marketBuyError);
    check('bonded.market_refuses_a_client_named_price',
      brandingRt.result.value.marketBuyRefusesAPrice === true, brandingRt.result.value.marketBuyError);

    // --- §23.5.5 VIEWER-SCOPED CARD DISPLAY: "player 1 can change how they see player 2's cards"
    // The rule is structural, so this probe drives the REAL API and then the REAL client module:
    // the layer is keyed by the viewer's own wallet, refuses a card-shaped request, writes no
    // branding binding, and the client paints a marked OPPONENT surface while leaving the marked
    // OWN surface alone (that is the scope the default setting promises).
    const cardViewRt = await cdp.send('Runtime.evaluate', {
      awaitPromise: true,
      expression: `(async () => {
        const out = { policyOk: false, scopes: 0, assetRequired: false, viewerScoped: false, cardsExcluded: false,
                      cardValueRefused: false, unknownFieldRefused: false, cardError: '', fieldError: '',
                      assetRequiredRefusal: false, foreignAssetRefused: false, assetError: '',
                      walletFieldRefused: false, walletFieldError: '',
                      capacityRefused: false, capacityError: '', capacityLimit: null, bindingsEmpty: true,
                      foreignPainted: 0, selfPainted: 0, art: '', observerOn: false, observerOff: false,
                      cleared: false, error: '' };
        const W = 'probe-cardview-wallet';
        // This block runs LAST, after every other assertion, so it shares an exhausted rate-limit
        // bucket. A 429 here is a limiter reading, not a product failure: retry with a real wait
        // (the app-entry mandate's "retry once, then report" rule, widened for a long probe).
        const sleep = ms => new Promise(r => setTimeout(r, ms));
        const jfetch = async (url, opts) => {
          let r = await fetch(url, opts);
          for (let i = 0; i < 3 && r.status === 429; i++) { await sleep(1500); r = await fetch(url, opts); }
          return r;
        };
        const post = async (wallet, body) => await (await jfetch('/api/assets/card-view?wallet=' + wallet, {
          method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(body) })).json();
        const pol = await (await jfetch('/api/assets/card-view')).json();
        const p = pol.policy || {};
        out.scopes = (p.scopes || []).length;
        out.assetRequired = p.asset_required === true;
        out.viewerScoped = p.viewer_scoped === true;
        out.cardsExcluded = p.cards_excluded === true;
        out.policyOk = pol.wallet_required === true && out.scopes >= 3 && out.assetRequired && out.viewerScoped && out.cardsExcluded;
        const cardVal = await post(W, { mode: 'deck_card' });
        out.cardError = String(cardVal.error || '');
        out.cardValueRefused = cardVal.success === false && out.cardError.indexOf('cards are excluded') === 0;
        const unknown = await post(W, { target_kind: 'deck_card', target_id: '1' });
        out.fieldError = String(unknown.error || '');
        out.unknownFieldRefused = unknown.success === false && out.fieldError.indexOf('invalid card-view body') === 0;
        // THEMING REQUIRES AN ASSET: asset mode with no asset is refused with the rule stated...
        const noAsset = await post(W, { mode: 'asset', scope: 'foreign_cards' });
        out.assetError = String(noAsset.error || '');
        out.assetRequiredRefusal = noAsset.success === false && out.assetError.indexOf('a bonded asset is required') === 0;
        // ...and an asset that is not the caller's own is refused too, so "any asset" is not enough.
        const foreignAsset = await post(W, { mode: 'asset', asset_id: 'BA-not-mine' });
        out.foreignAssetRefused = foreignAsset.success === false;
        // The caller cannot name another wallet: the accepted body has no field for one.
        const otherWallet = await post(W, { viewer_wallet: 'probe-cardview-other', mode: 'engine' });
        out.walletFieldError = String(otherWallet.error || '');
        out.walletFieldRefused = otherWallet.success === false && out.walletFieldError.indexOf('invalid card-view body') === 0;
        // CAPACITY, live: this wallet holds no card/deck NFTs, so creating a bonded asset is refused
        // with the card-supply rule and the served budget.
        const capMint = await (await jfetch('/api/assets/mint?wallet=' + W, {
          method: 'POST', headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ name: 'Probe art', asset_type: 0 }) })).json();
        out.capacityError = String(capMint.error || '');
        out.capacityRefused = capMint.success === false && out.capacityError.indexOf('capped at your card/deck NFT supply') >= 0;
        out.capacityLimit = capMint.capacity ? capMint.capacity.limit : null;
        // Pin the wallet FIRST, then ask the client which wallet it resolved: seeding one wallet
        // keeps the probe inside the limiter budget.
        window.currentWallet = W;
        const resolved = (typeof window.getActiveWallet === 'function' ? window.getActiveWallet() : '') || W;
        // A record can only exist for an asset this wallet OWNS, and this wallet has no card/deck NFT
        // supply with which to create one (the capacity rule, asserted above). The RENDER CONTRACT is
        // therefore driven white-box through the module's own exported state, while every SERVER
        // contract above was asserted through the real API - nothing is faked on the server side.
        const owned = await (await jfetch('/api/assets?wallet=' + W)).json();
        out.bindingsEmpty = !Array.isArray(owned.bindings) || owned.bindings.length === 0;
        if (window.CardViewSkins) {
          const st = window.CardViewSkins.state;
          st.loaded = true; st.active = true; st.mode = 'asset'; st.scope = 'foreign_cards';
          st.asset_id = 'BA-probe'; st.asset_name = 'Probe art';
          st.media = { uri: '/Assets/Images/NPC-helpers/Anya/Anya-001.png', mime_type: 'image/png' };
          out.art = window.CardViewSkins.artUri();
          const d = window.CardViewSkins.describe ? window.CardViewSkins.describe() : {};
          out.error = 'active=' + d.active + ' kind=' + (d.art_kind || '') + ' err=' + (d.error || '') + ' wallet=' + resolved;
          const host = document.createElement('div');
          host.innerHTML = '<div id="probe-cvs-foreign" data-card-owner="foreign"></div><div id="probe-cvs-self" data-card-owner="self"></div>';
          document.body.appendChild(host);
          window.CardViewSkins.applyTo(host);
          out.foreignPainted = document.querySelectorAll('#probe-cvs-foreign[data-cvs="on"]').length;
          out.selfPainted = document.querySelectorAll('#probe-cvs-self[data-cvs="on"]').length;
          out.observerOn = window.CardViewSkins.ensureObserver() === true;
          host.remove();
          // Restore the module to OFF so nothing after this point inherits a probe-only skin.
          st.active = false; st.mode = 'engine'; st.asset_id = ''; st.media = null; st.scope = 'foreign_cards';
          window.CardViewSkins.applyTo(document);
        } else { out.error = 'window.CardViewSkins missing'; }
        for (const w of [resolved, W]) {
          const cl = await (await jfetch('/api/assets/card-view/clear?wallet=' + w, { method: 'POST' })).json();
          if (w === resolved) out.cleared = cl.success === true;
        }
        if (window.CardViewSkins) {
          // The observer invariant is deterministic through the CLIENT's own clear: it applies the
          // server's answer and then re-evaluates whether an observer is needed at all. The wait is
          // deliberate — this probe drains a shared bucket (1 token/sec refill) and the client's
          // clear carries the standard one-shot retry, so give it headroom instead of racing it.
          await sleep(6000);
          try {
            await window.CardViewSkins.clear();
          } catch (e) {
            out.error += ' | clientClearErr=' + e.message;
          }
          out.observerOff = window.CardViewSkins.state.active === false && window.CardViewSkins.ensureObserver() === false;
        }
        // The server-side record is gone for this viewer. If the verification read is itself
        // rate-limited we must not pretend the clear failed: the server's clear response is the
        // authority, and the unverified read is reported in the detail instead.
        const afterClear = await (await jfetch('/api/assets/card-view?wallet=' + resolved)).json();
        if (afterClear && afterClear.view) {
          out.cleared = out.cleared === true && afterClear.view.active !== true;
        } else {
          out.cleared = out.cleared === true;
          out.error += ' | clearVerificationReadRefused';
        }
        // ── §10.6 slides + the placeholder pack, asserted through the REAL API ──────────────
        // The limiter is shared and refills 1 token/sec, and the blocks above have already spent it,
        // so wait for real headroom before the calls that matter. The pack's GRANT/REFUSAL arithmetic
        // (free, uncapped, idempotent, unknown/foreign ids refused, exact fee + sink routing) is
        // pinned deterministically by the Go tests; this block proves what only a browser can — that
        // the served pack, the slide vocabulary and the CLIENT's own calls actually work end to end.
        await sleep(9000);
        try {
          const cat = await (await jfetch('/api/assets/catalogue')).json();
          out.catalogueCount = Number(cat.count || 0);
          out.cataloguePrice = Number(cat.price_micro || 0);
          out.catalogueStarters = Array.isArray(cat.starter_skus) ? cat.starter_skus.length : 0;
          const skus = Array.isArray(cat.skus) ? cat.skus : [];
          // The rename, asserted from the SERVER's own view of the disk: every shipped frame is
          // addressed by a URL-safe path (the old "Anya (1).png" spelling would fail this).
          out.catalogueUrlsSafe = skus.length > 0 && skus.every(s => !/[ ()?#]/.test(String(s.uri || '')));
          out.catalogueOneUnavailable = skus.some(s => s.purchasable === false && !!s.unavailable);
        } catch (e) { out.catalogueError = e.message; }
        try {
          const g1 = await (await jfetch('/api/assets/starter?wallet=' + W, { method: 'POST' })).json();
          const granted = Array.isArray(g1.granted) ? g1.granted : [];
          const already = Array.isArray(g1.already_owned) ? g1.already_owned : [];
          // A repeat run of this probe finds the pack ALREADY granted, which is the idempotency
          // contract showing itself: what matters is that the wallet HOLDS the six frames.
          out.starterHeld = granted.length + already.length;
          out.starterSkipped = Array.isArray(g1.skipped) ? g1.skipped.length : -1;
          // FREE and UNCAPPED: held by a wallet whose card/deck NFT supply is 0.
          out.starterFree = g1.success === true && !!g1.capacity && Number(g1.capacity.limit) === 0;
          out.starterAssetId = granted.length ? String(granted[0].asset_id || '')
            : (already.length ? String(already[0].asset_id || '') : '');
        } catch (e) { out.starterError = e.message; }
        // One write is enough to prove the client→server→surface chain, and the bucket refills at
        // 1 token/sec, so wait for it rather than racing (the pack's refusal arithmetic is pinned
        // deterministically by the Go tests, which do not share this limiter).
        await sleep(11000);
        if (window.SlideTheming) {
          try {
            const d0 = await window.SlideTheming.refresh({ force: true });
            const pol = (d0 && d0.policy) || {};
            out.slideSlots = Array.isArray(pol.slots) ? pol.slots.length : 0;
            out.slideAssetRequired = pol.asset_required === true;
            // No served KEY may name a card (values may state the rule).
            out.slideKeysCardFree = !/"[a-z_]*card[a-z_]*"\s*:/i.test(JSON.stringify(pol));
            out.slideDefaultsReal = Array.isArray(pol.slots) && pol.slots.every(s => !!s.default_uri);
            out.slideStarterIsDefault = Array.isArray(pol.slots) && pol.slots.every(s => String(s.default_sku || '').length > 0);
            out.slideRefusalRuleServed = String((pol.rule || '')).length > 0 && pol.asset_required === true;
            if (out.starterAssetId) {
              const d1 = await window.SlideTheming.setSlot('menu_slide_1', out.starterAssetId);
              out.slideActive = !!d1 && d1.themed === 1;
              const views = window.SlideTheming.slotViews();
              const slot = views.filter(v => v.slot === 'menu_slide_1')[0] || {};
              out.slidePainted = slot.art_kind === 'asset' && !!slot.art_uri;
              out.slideArt = String(slot.art_uri || '');
            }
          } catch (e) { out.slideError = e.message; }
          // The renderer: three layers, all painted, exactly one faded in — then cleaned up.
          const host = document.createElement('div');
          host.style.cssText = 'position:fixed;inset:0;';
          document.body.appendChild(host);
          window.SlideTheming.mount('menu', host);
          window.SlideTheming.paint('menu');
          const layers = host.querySelectorAll('.slide-theme-layer');
          out.slideLayers = layers.length;
          out.slideLayersPainted = Array.prototype.filter.call(layers, l => !!l.style.backgroundImage).length;
          out.slideOneActive = host.querySelectorAll('.slide-theme-layer.is-active').length;
          window.SlideTheming.unmount('menu');
          host.remove();
          // The APP's own boot screen mounts the same show — this asserts the WIRING (main menu),
          // not merely the renderer. Only the first .constellation-bg is inspected so a call here can
          // never double-count a menu the app already built.
          try {
            if (typeof window.initMenuConstellation === 'function') window.initMenuConstellation();
            const bg0 = document.querySelectorAll('.constellation-bg')[0];
            out.bootSlideLayers = bg0 ? bg0.querySelectorAll('.slide-theme-layer').length : 0;
            out.bootSlidePainted = bg0
              ? Array.prototype.filter.call(bg0.querySelectorAll('.slide-theme-layer'), l => !!l.style.backgroundImage).length
              : 0;
          } catch (e) { out.bootSlideErr = e.message; }
          try { await window.SlideTheming.clearSlot('menu_slide_1'); } catch (e) { /* reported by the module */ }
        } else { out.slideError = 'window.SlideTheming missing'; }
        window.currentWallet = undefined;
        return out;
      })()`,
      returnByValue: true,
    });
    const cv = cardViewRt.result.value;
    check('bonded.card_view_policy_served', cv.policyOk === true, 'scopes=' + cv.scopes + ' assetRequired=' + cv.assetRequired + ' viewerScoped=' + cv.viewerScoped + ' cardsExcluded=' + cv.cardsExcluded);
    check('bonded.card_view_refuses_card_request', cv.cardValueRefused === true && cv.unknownFieldRefused === true, cv.cardError + ' | ' + cv.fieldError);
    check('bonded.card_view_requires_a_bonded_asset', cv.assetRequiredRefusal === true && cv.foreignAssetRefused === true, cv.assetError);
    check('bonded.card_view_cannot_name_another_wallet', cv.walletFieldRefused === true, cv.walletFieldError);
    check('bonded.capacity_caps_bonded_asset_creation', cv.capacityRefused === true && cv.capacityLimit === 0, cv.capacityError + ' limit=' + cv.capacityLimit);
    check('bonded.card_view_writes_no_branding', cv.bindingsEmpty === true, 'bindingsEmpty=' + cv.bindingsEmpty);
    check('bonded.card_view_client_honours_scope', cv.foreignPainted === 1 && cv.selfPainted === 0, 'foreign=' + cv.foreignPainted + ' self=' + cv.selfPainted + ' art=' + cv.art + ' err=' + cv.error);
    check('bonded.card_view_observer_only_while_active', cv.observerOn === true && cv.observerOff === true, 'on=' + cv.observerOn + ' offAfterClear=' + cv.observerOff);
    check('bonded.card_view_clears', cv.cleared === true, 'cleared=' + cv.cleared);
    check('sandbox.pack_catalogue_served', cv.catalogueCount >= 100 && cv.cataloguePrice > 0 && cv.catalogueStarters >= 1 && cv.catalogueUrlsSafe === true, 'count=' + cv.catalogueCount + ' price=' + cv.cataloguePrice + ' starters=' + cv.catalogueStarters + ' urlsSafe=' + cv.catalogueUrlsSafe + ' err=' + (cv.catalogueError || ''));
    check('sandbox.pack_reports_an_unavailable_frame', cv.catalogueOneUnavailable === true, 'unavailable=' + cv.catalogueOneUnavailable);
    check('sandbox.starter_pack_is_free_and_uncapped', cv.starterHeld >= 6 && cv.starterFree === true && cv.starterSkipped === 0, 'held=' + cv.starterHeld + ' free=' + cv.starterFree + ' skipped=' + cv.starterSkipped + ' err=' + (cv.starterError || ''));
    check('sandbox.slide_policy_served', cv.slideSlots === 6 && cv.slideAssetRequired === true && cv.slideDefaultsReal === true && cv.slideStarterIsDefault === true, 'slots=' + cv.slideSlots + ' assetRequired=' + cv.slideAssetRequired + ' defaults=' + cv.slideDefaultsReal + ' err=' + (cv.slideError || ''));
    check('sandbox.slide_vocabulary_is_card_free', cv.slideKeysCardFree === true, 'cardFree=' + cv.slideKeysCardFree);
    check('sandbox.slide_rule_is_served_with_the_policy', cv.slideRefusalRuleServed === true, 'ruleServed=' + cv.slideRefusalRuleServed);
    check('sandbox.slide_wears_your_own_asset', cv.slideActive === true && cv.slidePainted === true, 'active=' + cv.slideActive + ' painted=' + cv.slidePainted + ' art=' + cv.slideArt);
    check('sandbox.slideshow_renders_three_layers', cv.slideLayers === 3 && cv.slideLayersPainted === 3 && cv.slideOneActive === 1, 'layers=' + cv.slideLayers + ' painted=' + cv.slideLayersPainted + ' active=' + cv.slideOneActive);
    check('sandbox.boot_screen_slideshow_mounted', cv.bootSlideLayers === 3 && cv.bootSlidePainted === 3, 'layers=' + cv.bootSlideLayers + ' painted=' + cv.bootSlidePainted + ' err=' + (cv.bootSlideErr || ''));

    // --- §10.6/§10.7 THE LIGHT RENDITIONS + THE PER-TREE CHROME ART -------------------------------
    // Two claims only a browser + the real API can settle: (1) the light art the server derives is
    // really ON THE WIRE (fetched and measured against the byte size the payload recorded for it),
    // and (2) the World Dashboard's marked trees are really PAINTED by the client module.
    const artRt = await cdp.send('Runtime.evaluate', {
      awaitPromise: true,
      expression: `(async () => {
        const sleep = ms => new Promise(r => setTimeout(r, ms));
        const out = { ready: false, derived: 0, total: 0, failed: 0, reusedField: true, statusError: '',
                      artStatus: 0, artBytes: 0, artRecorded: 0, artLighter: false, artUri: '',
                      treeCount: 0, treeUnique: false, treePinned: '', treeAllHaveArt: false, treeDerived: 0,
                      treeError: '', marked: 0, painted: 0, cssPainted: 0, missing: 0, treeSlides: 0,
                      paintError: '' };
        // This block runs LAST, when the shared 1-token/sec bucket is at its emptiest, so every API
        // read retries a 429 (a limiter reading is not a product failure).
        const jf = async (url, opts) => {
          let r = await fetch(url, opts);
          for (let i = 0; i < 3 && r.status === 429; i++) { await sleep(1500); r = await fetch(url, opts); }
          return r;
        };
        await sleep(4000);
        try {
          const cat = await (await jf('/api/assets/catalogue')).json();
          const st = cat.derivative_status || {};
          out.ready = st.ready === true;
          out.derived = Number(st.derived || 0);
          out.total = Number(st.total || 0);
          out.failed = Number(st.failed || 0);
          // A resting-state census cannot know a per-pass fact, so the payload must not claim one.
          out.reusedField = Object.prototype.hasOwnProperty.call(st, 'reused');
          // The thumb is REAL art: fetch it and compare with the recorded size.
          const skus = Array.isArray(cat.skus) ? cat.skus : [];
          const withThumb = skus.filter(s => s.derivatives && s.derivatives.thumb && s.derivatives.thumb.uri)[0];
          if (withThumb) {
            const rec = withThumb.derivatives.thumb;
            const r = await fetch(rec.uri, { cache: 'no-store' });
            const buf = await r.arrayBuffer();
            out.artStatus = r.status;
            out.artBytes = buf.byteLength;
            out.artRecorded = Number(rec.bytes || 0);
            out.artLighter = buf.byteLength < Number(withThumb.bytes || 0);
            out.artUri = String(rec.uri || '');
          }
        } catch (e) { out.statusError = e.message; }
        try {
          const ut = await (await jf('/api/assets/ui-trees')).json();
          out.treeCount = Number(ut.count || 0);
          out.treeUnique = ut.unique === true;
          out.treePinned = (ut.pinned_trees || []).map(p => String(p.tree_id) + '=' + String(p.character)).join(',');
          const trees = Array.isArray(ut.trees) ? ut.trees : [];
          out.treeAllHaveArt = trees.length > 0 && trees.every(t => !!t.uri && !!t.source_uri);
          out.treeDerived = trees.filter(t => t.derived === true).length;
        } catch (e) { out.treeError = e.message; }
        // The CLIENT half: the dashboard's marked trees must actually be painted.
        try {
          if (typeof window.openWorldDashboard === 'function') window.openWorldDashboard();
          await sleep(700);
          if (window.SlideTheming && window.SlideTheming.loadUiTrees) {
            await window.SlideTheming.loadUiTrees(true);
            window.SlideTheming.paintUiTrees(document);
            const d = window.SlideTheming.describe();
            out.marked = d.uiTrees.marked;
            out.painted = d.uiTrees.painted;
            out.missing = (d.uiTrees.missing || []).length;
            out.treeSlides = d.derivedSlides;
          }
          const marked = document.querySelectorAll('[data-wd-tree]');
          out.cssPainted = Array.prototype.filter.call(marked,
            el => !!el.style.getPropertyValue('--wd-tree-art')).length;
          if (typeof window.hideAllOverlays === 'function') window.hideAllOverlays();
        } catch (e) { out.paintError = e.message; }
        return out;
      })()`,
      returnByValue: true,
    });
    const av = artRt.result.value;
    check('sandbox.light_renditions_are_real_art',
      av.ready === true && av.derived === av.total && av.total >= 100 && av.failed === 0 &&
      av.artStatus === 200 && av.artBytes === av.artRecorded && av.artLighter === true,
      'derived=' + av.derived + '/' + av.total + ' status=' + av.artStatus + ' bytes=' + av.artBytes +
      ' recorded=' + av.artRecorded + ' lighter=' + av.artLighter + ' uri=' + av.artUri + ' err=' + av.statusError);
    check('sandbox.derivative_status_claims_no_pass_fact', av.reusedField === false, 'hasReusedField=' + av.reusedField);
    check('sandbox.slides_show_the_light_rendition', av.treeSlides >= 6, 'derivedSlides=' + av.treeSlides);
    check('sandbox.every_ui_tree_has_its_own_frame',
      av.treeCount >= 60 && av.treeUnique === true && av.treeAllHaveArt === true &&
      av.treePinned === 'rewards=Crypto-seraph,achievements=Crypto-seraph',
      'count=' + av.treeCount + ' unique=' + av.treeUnique + ' pinned=' + av.treePinned +
      ' allHaveArt=' + av.treeAllHaveArt + ' derived=' + av.treeDerived + ' err=' + av.treeError);
    check('sandbox.dashboard_paints_the_tree_art',
      av.marked > 0 && av.painted > 0 && av.missing === 0 && av.cssPainted === av.painted,
      'marked=' + av.marked + ' painted=' + av.painted + ' cssPainted=' + av.cssPainted +
      ' missing=' + av.missing + ' err=' + av.paintError);

    // --- Admin Suite: declared controls, wired listeners, published globals -----
    // The panel's markup used to call `adminX()` from inline `onclick` attributes. An inline handler
    // resolves the name on `window`, and EIGHT of these functions were published by nothing — so
    // those controls threw ReferenceError and did nothing. This asserts the replacement: the markup
    // DECLARES (`data-admin-action`), the module BINDS (proved by driving a control and reading its
    // DOM effect), and the suite's functions exist for the rows admin.js renders itself.
    const adminSuite = await cdp.send('Runtime.evaluate', {
      expression: `(async () => {
        const out = { panel: false, declared: 0, inline: 0, globalsMissing: [], wired: false, labelAfter: '' };
        const root = document.getElementById('admin-control-panel');
        if (!root) return out;
        out.panel = true;
        out.declared = root.querySelectorAll('[data-admin-action]').length;
        out.inline = root.querySelectorAll('[onclick],[oninput],[onchange]').length;
        ['adminUpdatePowerScaling','adminBanWallet','adminAvatarBan','adminResetStats',
         'adminUpdateDLCProduct','fetchDLCRegistry','fetchAdminLogs','renderShopTokenPresetEditor',
         'saveShopTokenPresetsFromEditor'].forEach(name => {
          if (typeof window[name] !== 'function') out.globalsMissing.push(name);
        });
        // BINDING PROOF: drive the base-power slider. Its read-out is updated ONLY by the bound
        // listener (the inline handler that used to do it is gone), so a stale read-out means the
        // control is not wired at all. The module reports a handler's REJECTION, so an action runs
        // through a promise — hence the awaited tick before the read-out is inspected.
        const slider = document.getElementById('admin-power-base');
        const label = document.getElementById('power-base-val');
        if (slider && label) {
          slider.value = '123';
          slider.dispatchEvent(new Event('input', { bubbles: true }));
          await new Promise((r) => setTimeout(r, 0));
          out.labelAfter = label.innerText.trim();
          out.wired = out.labelAfter === '123';
        }
        return out;
      })()`,
      returnByValue: true,
      awaitPromise: true,
    });
    const as = adminSuite.result.value;
    check('admin.suite_controls_are_declared_not_inline',
      as.panel === true && as.declared >= 24 && as.inline === 0,
      'declared=' + as.declared + ' inlineHandlers=' + as.inline);
    check('admin.suite_controls_are_wired_by_the_module', as.wired === true, 'labelAfter=' + as.labelAfter);
    check('admin.suite_globals_are_published', as.globalsMissing.length === 0, JSON.stringify(as.globalsMissing));

    // --- Tenant world lease: the served catalogue reaches the panel -----------------
    // The infrastructure panel read `res.leases[].price_micro` — a shape `/api/lease/available` has
    // never served — so it could only answer "No leases available", whatever the server said. These
    // assert (1) the served shape and (2) that the panel renders the catalogue AND states why a
    // lease cannot be created yet. A rate-limited read is RETRIED once, never counted as a pass.
    const leaseRt = await cdp.send('Runtime.evaluate', {
      expression: `(async () => {
        const out = { status: 0, systems: 0, creationAvailable: null, blockedBy: 0, hasLeasesKey: true,
                      panelText: '', stated: false, tried: 0 };
        const base = (window.CONFIG && CONFIG.API_BASE) ? CONFIG.API_BASE : '';
        for (let attempt = 0; attempt < 3 && out.status !== 200; attempt++) {
          out.tried = attempt + 1;
          try {
            const r = await fetch(base + '/api/lease/available');
            out.status = r.status;
            if (r.status === 200) {
              const j = await r.json();
              out.systems = Array.isArray(j.systems) ? j.systems.length : -1;
              out.creationAvailable = j.creation_available;
              out.blockedBy = Array.isArray(j.creation_blocked_by) ? j.creation_blocked_by.length : -1;
              out.hasLeasesKey = Object.prototype.hasOwnProperty.call(j, 'leases');
            }
          } catch (e) { out.error = e.message; }
          if (out.status !== 200) await new Promise((res) => setTimeout(res, 1200));
        }

        if (typeof window.switchWDTab === 'function') window.switchWDTab('infrastructure');
        for (let attempt = 0; attempt < 3; attempt++) {
          await new Promise((res) => setTimeout(res, 900));
          const el = document.getElementById('wd-infrastructure');
          out.panelText = (el && el.textContent) ? el.textContent.trim() : '';
          if (out.panelText.includes('Leasing is not available yet')) { out.stated = true; break; }
        }
        if (typeof window.hideAllOverlays === 'function') window.hideAllOverlays();
        return out;
      })()`,
      returnByValue: true,
      awaitPromise: true,
    });
    const lr = leaseRt.result.value;
    check('lease.catalogue_serves_the_shape_the_client_reads',
      lr.status === 200 && lr.systems > 0 && lr.creationAvailable === false && lr.blockedBy === 5 &&
      lr.hasLeasesKey === false,
      'status=' + lr.status + ' systems=' + lr.systems + ' creationAvailable=' + lr.creationAvailable +
      ' blockedBy=' + lr.blockedBy + ' servedLeasesKey=' + lr.hasLeasesKey + ' tried=' + lr.tried + ' err=' + (lr.error || ''));
    check('lease.panel_states_why_a_lease_cannot_be_created',
      lr.stated === true,
      'panelText=' + (lr.panelText || '').slice(0, 160));

    // --- THE MENU WHEEL: one taxonomy, a hub-and-spokes layout, and it SPINS ----
    // The wheel was ~80% built but (a) the category list it rendered had rivals,
    // (b) it could not be spun or discovered, and (c) the panel that owns it styled
    // ONLY its controller tab. Each claim below is a measurement of the live app.
    const wheelRt = await cdp.send('Runtime.evaluate', {
      expression: `(async () => {
        const out = {};
        const U = window.UserPreferences;
        const MC = window.MenuCustomization;
        const tax = window.WDTaxonomy;

        // 1. ONE taxonomy, published READ-ONLY.
        out.categoryCount = tax ? tax.categories.length : -1;
        out.featureCount = (tax && tax.featureTabs) ? tax.featureTabs().length : -1;
        out.frozenTop = tax ? Object.isFrozen(tax) : false;
        out.frozenCats = (tax && tax.categories && tax.categories[0]) ? Object.isFrozen(tax.categories[0]) : false;
        out.mutationRefused = false;
        try { tax.categories.push({ id: 'nope' }); } catch (e) { out.mutationRefused = true; }

        // 2. The grid library is the ONE owner of the layout classification.
        out.hasSpokes = !!(MC && MC.GRIDS && MC.GRIDS.spokes);
        out.spokesIsRadial = !!(MC && MC.isRadialGrid && MC.isRadialGrid('spokes'));
        out.linearIsFlow = !!(MC && MC.isPositionedGrid && MC.isPositionedGrid('linearV') === false);
        out.gridIsFlow = !!(MC && MC.isPositionedGrid && MC.isPositionedGrid('grid') === false);
        out.circleIsPositioned = !!(MC && MC.isPositionedGrid && MC.isPositionedGrid('circle') === true);

        // 3. Hub-and-spokes geometry: item 0 IS the hub, the rest ride the ring.
        const items = [{ size: 'md' }, { size: 'md' }, { size: 'md' }];
        const pos = MC.computePositions('spokes', items, 400, 300, { gridRadius: '40%', gridRotation: '0deg' });
        out.posCount = pos.length;
        out.hubCentered = Math.abs((pos[0].x + pos[0].w / 2) - 200) < 1 && Math.abs((pos[0].y + pos[0].h / 2) - 150) < 1;
        out.ringOffCentre = Math.hypot((pos[1].x + pos[1].w / 2) - 200, (pos[1].y + pos[1].h / 2) - 150) > 50;
        // 4. Star a category, choose spokes, render, SPIN, and prove it persisted.
        if (!U.isStarred('player', '')) U.toggleStar('player', '', 'Player Hub', '🎮', '#00bcd4');
        U.setMenuLayout({ gridType: 'spokes', gridRadius: '40%', gridSpin: 0 });
        window.renderStarredItems();
        await new Promise(r => setTimeout(r, 150));
        const grid = document.getElementById('ch-starred-grid');
        out.buttons = grid ? grid.querySelectorAll('.ch-starred-btn').length : -1;
        window.spinMenuWheel(45);
        out.gridTransform = grid ? (grid.style.transform || '') : 'MISSING';
        const firstBtn = grid ? grid.querySelector('.ch-starred-btn') : null;
        out.itemTransform = firstBtn ? (firstBtn.style.transform || '') : 'MISSING';
        out.persistedSpin = U.getMenuLayout().gridSpin;
        out.hintVisible = (() => { const h = document.getElementById('ch-starred-hint'); return h ? getComputedStyle(h).display !== 'none' : false; })();

        // 5. A LAYOUT-ONLY change must re-render (the guard used to skip exactly this,
        //    so the panel's own Save looked like it had not been saved).
        U.setMenuLayout({ gridType: 'grid' });
        window.renderStarredItems();
        await new Promise(r => setTimeout(r, 120));
        out.afterFlowRender = grid ? (grid.style.transform || 'NONE') : 'MISSING';
        U.setMenuLayout({ gridType: 'spokes' });
        window.spinMenuWheel(30);
        out.reRenderedSpin = grid ? (grid.style.transform || 'NONE') : 'MISSING';
        window.resetMenuSpin();

        // 6. The panel: spokes is offered, the spin row appears AND is styled, and the
        //    Customize affordance is labelled.
        if (typeof window.openMenuCustomization === 'function') window.openMenuCustomization();
        await new Promise(r => setTimeout(r, 220));
        out.spokesOption = !!document.querySelector('.mc-grid-option[data-grid="spokes"]');
        out.gridOptionCount = document.querySelectorAll('.mc-grid-option').length;
        const gridTab = document.querySelector('.mc-tab[data-tab="grid"]');
        if (gridTab) gridTab.click();
        await new Promise(r => setTimeout(r, 150));
        const spinRow = document.getElementById('mc-spin-row');
        out.spinRowDisplay = spinRow ? getComputedStyle(spinRow).display : 'MISSING';
        const customize = document.querySelector('.customize-menu-btn');
        out.customizeLabelled = customize ? /Customize/i.test(customize.textContent || '') : false;
        if (typeof window.hideAllOverlays === 'function') window.hideAllOverlays();

        // 7. The DEV/GAME HUB: reachable from the ONE navigation surface, and it must
        //    NOT report a purchase it cannot make (it used to toast "✅ Purchased").
        out.devhubFeature = !!(tax && tax.featureTabs && tax.featureTabs().indexOf('devhub') >= 0);
        out.devhubRoutesOut = !!(tax && tax.categories && tax.categories.some(function (c) {
          return c.features.some(function (f) { return f.tab === 'devhub' && f.routes_out === true; });
        }));
        out.devhubOpenerPublished = typeof window.openDevGameHub === 'function';
        if (out.devhubOpenerPublished) {
          window.openDevGameHub();
          await new Promise(r => setTimeout(r, 220));
          out.dghItems = document.querySelectorAll('#dgh-catalog .dgh-item').length;
          const firstBuy = document.querySelector('#dgh-catalog .dgh-buy');
          if (firstBuy) firstBuy.click();
          await new Promise(r => setTimeout(r, 150));
          const st = document.getElementById('dgh-status');
          out.dghStatus = st ? (st.textContent || '') : 'MISSING';
          out.dghClaimsPurchase = !/cannot be purchased yet/i.test(out.dghStatus);
          const ownedEl = document.getElementById('dgh-owned');
          out.dghOwned = ownedEl ? ownedEl.textContent : 'MISSING';
          if (typeof window.hideAllOverlays === 'function') window.hideAllOverlays();
        }

        // 8. THE §27 THEME CONTRACT. theme_engine.js was the ONE baseline entry whose verdict was
        //    WIRE (not retire), so app.js now composes it. The SCSS side already existed
        //    (_theme_engine.scss + _variables.scss, consumed by 10 partials) and was INERT: nothing
        //    else in Public/**/*.js writes --theme-accent*. This measures the contract end to end —
        //    published, DRIVING the tokens, and defaulting to the SAME element the compiled :root
        //    ships (so composing it activates the contract instead of changing the look).
        out.themePublished = !!(window.themeEngine && typeof window.themeEngine.setElement === 'function');
        const accentOf = () => (getComputedStyle(document.documentElement).getPropertyValue('--theme-accent') || '').trim();
        out.themeAccentDefault = accentOf();
        out.themeElementDefault = out.themePublished ? window.themeEngine.getElement() : 'MISSING';
        if (out.themePublished) {
          window.themeEngine.setElement('water');
          out.themeAccentWater = accentOf();
          out.themeRgbWater = (getComputedStyle(document.documentElement).getPropertyValue('--theme-accent-rgb') || '').trim();
          window.themeEngine.setElement('fire'); // restore the shipped default
        }
        out.themeAccentAfterReset = accentOf();
        // 8b. THE SERVED VECTOR FEEDS THE CONTRACT. theme_engine.js now reads /api/theme/vector
        //     (handleThemeVector) and applies the engine's element/intensity, so the palette is fed
        //     by §27 rather than by localStorage alone. A refused read must KEEP the last theme and
        //     record data-theme-source="fallback" rather than silently reverting to a default.
        out.themeServerVectorFn = out.themePublished && typeof window.themeEngine.applyServerVector === 'function';
        out.themeElementNames = out.themePublished ? (window.themeEngine.elements || []).length : 0;
        out.themeSource = document.documentElement.getAttribute('data-theme-source') || '';

        // 8c. THE FIVE OPERATIONS CONSOLES. Each was composed by app.js and published every handler
        //     name it renders, yet NONE could render: the root container it looks up
        //     (#<x>-dashboard-overlay) exists in no markup and the module never created it, so init()
        //     returned on the null-check and the boot-time guard never fired. They now self-mount.
        //     This drives each one through its own opener and reads the COMPUTED result.
        out.consoles = {};
        for (const [name, opener, rootId] of [
          ['community', 'openCommunityDashboard', 'community-dashboard-overlay'],
          ['extended', 'openExtendedDashboard', 'extended-dashboard-overlay'],
          ['security', 'openSecurityDashboard', 'security-dashboard-overlay'],
          ['system', 'openSystemDashboard', 'system-dashboard-overlay'],
          ['utilities', 'openUtilitiesDashboard', 'utilities-dashboard-overlay'],
        ]) {
          const rec = { published: typeof window[opener] === 'function', root: false, computed: 'MISSING', panels: 0 };
          if (rec.published) {
            try {
              window[opener]();
              await new Promise(r => setTimeout(r, 180));
              const el = document.getElementById(rootId);
              rec.root = !!el;
              rec.computed = el ? (getComputedStyle(el).display || '') : 'MISSING';
              rec.panels = el ? el.querySelectorAll('.cd-tab, .ed-tab, .sd-tab, .ud-tab').length : 0;
            } catch (e) { rec.error = String(e && e.message || e); }
            if (typeof window.hideAllOverlays === 'function') window.hideAllOverlays();
          }
          out.consoles[name] = rec;
        }

        // 8d. THE NINE NEW LEAVES. Every one is a surface that was COMPOSED and opened by NOTHING;
        //     each now has a World Dashboard leaf, and the ones with a full-screen owner must route
        //     out to a PUBLISHED opener (a leaf that names a missing function embeds silently).
        //     NOTE: 'tax' is declared once at the top of this block — redeclaring it here would be a
        //     SyntaxError that kills the ENTIRE eval (and the failure looks like a missing return).
        const catOf = (id) => ((tax && tax.categories) || []).find(c => c.id === id);
        const tabsOf = (id) => { const c = catOf(id); return c ? c.features.map(f => f.tab) : []; };
        out.newLeaves = {
          theme: tabsOf('player').includes('theme'),
          faction: tabsOf('careers').includes('faction'),
          religion: tabsOf('faith').includes('religion'),
          bridge: tabsOf('economy').includes('bridge'),
          consoles: ['community', 'exconsole', 'secconsole', 'sysconsole', 'utconsole']
            .every(t => tabsOf('system').includes(t)),
        };
        out.newLeafOpenersPublished = {
          theme: typeof window.openThemeDashboard === 'function',
          religion: typeof window.openReligionGovernance === 'function',
          bridge: typeof window.openBridgeRouter === 'function',
          faction: typeof window.initFactionShop === 'function',
        };
        // 2026-09-15 (g): the CRIMINALITY HUB + the Intel-Agent cyber-intercept door. The hub exists
        // because the World Dashboard's leaf used to BE openCourthouse, which returns early with no
        // Wanted Level — a button that silently no-opped for every clean player.
        (function () {
          try {
            var st = {};
            try { st = window.GetGameState() || {}; } catch (e0) { st = {}; }
            out.criminalityWanted = Number(st.wanted_level || 0);
            window.openCriminality();
            var hub = document.getElementById('criminality-hub-overlay');
            var btns = hub ? Array.prototype.slice.call(hub.querySelectorAll('button')) : [];
            out.criminalityHub = !!hub;
            out.criminalitySections = btns.map(function (b) {
              return (b.textContent || '').replace(/[\s]+/g, ' ').trim().slice(0, 34);
            });
            out.criminalityCourthouseDisabled = btns.some(function (b) { return b.disabled === true; });
            if (hub) hub.remove();
            window.openCyberIntercept();
            var cyb = document.getElementById('cyber-intercept-overlay');
            out.cyberInterceptOverlay = !!cyb && !!document.getElementById('cyber-intercept-scan');
            out.cyberInterceptPublished = typeof window.submitCyberIntercept === 'function';
            if (cyb) cyb.remove();
          } catch (e1) { out.criminalityError = String(e1 && e1.message); }
        })();
        return out;
      })()`,
      returnByValue: true,
      awaitPromise: true,
    });
    const wheel = wheelRt.result.value;
    check('menu.one_taxonomy_published_read_only',
      // The feature count is a DRIFT GUARD, not a magic number: it must be raised deliberately when a
      // leaf is added, which is the point (a silently shrinking wheel is the failure being caught).
      // 68 → 77 on 2026-09-15 (f): theme, faction, bridge, religion + the five operations consoles
      // (community/exconsole/secconsole/sysconsole/utconsole) each gained a leaf, because every one
      // of those surfaces was built and reachable by nothing.
      wheel.categoryCount === 11 && wheel.featureCount === 77 && wheel.frozenTop === true &&
      wheel.frozenCats === true && wheel.mutationRefused === true,
      'categories=' + wheel.categoryCount + ' features=' + wheel.featureCount +
      ' frozen=' + wheel.frozenTop + '/' + wheel.frozenCats + ' mutationRefused=' + wheel.mutationRefused);
    check('menu.hub_and_spokes_layout_exists',
      wheel.hasSpokes === true && wheel.spokesIsRadial === true && wheel.posCount === 3 &&
      wheel.hubCentered === true && wheel.ringOffCentre === true,
      'spokes=' + wheel.hasSpokes + ' radial=' + wheel.spokesIsRadial + ' pos=' + wheel.posCount +
      ' hubCentered=' + wheel.hubCentered + ' ringOffCentre=' + wheel.ringOffCentre);
    check('menu.layout_classification_has_one_owner',
      wheel.linearIsFlow === true && wheel.gridIsFlow === true && wheel.circleIsPositioned === true,
      'linearV_flow=' + wheel.linearIsFlow + ' grid_flow=' + wheel.gridIsFlow +
      ' circle_positioned=' + wheel.circleIsPositioned);
    check('menu.wheel_spins_and_persists',
      /rotate\(45deg\)/.test(wheel.gridTransform) && /rotate\(-45deg\)/.test(wheel.itemTransform) &&
      wheel.persistedSpin === 45 && wheel.hintVisible === true,
      'grid=' + wheel.gridTransform + ' item=' + wheel.itemTransform +
      ' persisted=' + wheel.persistedSpin + ' hint=' + wheel.hintVisible);
    check('menu.layout_change_re_renders_without_a_reload',
      wheel.afterFlowRender === 'NONE' && /rotate\(30deg\)/.test(wheel.reRenderedSpin) && wheel.buttons > 0,
      'afterFlow=' + wheel.afterFlowRender + ' afterSpin=' + wheel.reRenderedSpin +
      ' buttons=' + wheel.buttons);
    check('menu.customization_panel_offers_and_styles_the_wheel',
      wheel.spokesOption === true && wheel.gridOptionCount >= 10 && wheel.spinRowDisplay === 'flex' &&
      wheel.customizeLabelled === true,
      'spokesOption=' + wheel.spokesOption + ' options=' + wheel.gridOptionCount +
      ' spinRowDisplay=' + wheel.spinRowDisplay + ' labelled=' + wheel.customizeLabelled);
    check('devhub.is_reachable_and_honest_about_purchasing',
      wheel.devhubFeature === true && wheel.devhubRoutesOut === true && wheel.devhubOpenerPublished === true &&
      wheel.dghItems > 0 && wheel.dghClaimsPurchase === false && wheel.dghOwned === '0',
      'feature=' + wheel.devhubFeature + ' routesOut=' + wheel.devhubRoutesOut +
      ' opener=' + wheel.devhubOpenerPublished + ' items=' + wheel.dghItems +
      ' claimsPurchase=' + wheel.dghClaimsPurchase + ' owned=' + wheel.dghOwned +
      ' status=' + String(wheel.dghStatus || '').slice(0, 120));
    check('theme.client_contract_is_composed_and_drives_the_tokens',
      wheel.themePublished === true && wheel.themeElementDefault === 'fire' &&
      /#ff6b35/i.test(wheel.themeAccentDefault) && /#00d4ff/i.test(wheel.themeAccentWater) &&
      String(wheel.themeRgbWater).replace(/\s/g, '') === '0,212,255' &&
      /#ff6b35/i.test(wheel.themeAccentAfterReset),
      'published=' + wheel.themePublished + ' element=' + wheel.themeElementDefault +
      ' default=' + wheel.themeAccentDefault + ' water=' + wheel.themeAccentWater +
      ' rgbWater=' + wheel.themeRgbWater + ' afterReset=' + wheel.themeAccentAfterReset);

    // ── 2026-09-15 (f) PLACEMENT PASS ────────────────────────────────────────────────────────────
    // The palette is now fed by the ENGINE, not by localStorage alone.
    check('theme.palette_is_fed_by_the_served_vector',
      wheel.themeServerVectorFn === true && wheel.themeElementNames === 7,
      'applyServerVector=' + wheel.themeServerVectorFn + ' elementNames=' + wheel.themeElementNames +
      ' source=' + wheel.themeSource);
    // The five consoles: reachable, self-mounted, and actually RENDERING (a root that exists but
    // computes to none would be the same silent failure the fix removed).
    const consoleNames = ['community', 'extended', 'security', 'system', 'utilities'];
    const consoleOk = consoleNames.every(n => {
      const r = (wheel.consoles || {})[n] || {};
      return r.published === true && r.root === true && r.computed === 'flex' && r.panels > 0;
    });
    check('console.five_operations_consoles_are_dead_no_more',
      consoleOk,
      consoleNames.map(n => {
        const r = (wheel.consoles || {})[n] || {};
        return n + '(pub=' + r.published + ',root=' + r.root + ',computed=' + r.computed + ',tabs=' + r.panels + ')';
      }).join(' '));
    // Every surface that was composed-and-opened-by-nothing now has a leaf, and each owner that
    // needed one publishes the opener its leaf names.
    const nl = wheel.newLeaves || {};
    check('wd.the_unreachable_surfaces_now_have_leaves',
      nl.theme === true && nl.faction === true && nl.religion === true && nl.bridge === true && nl.consoles === true,
      JSON.stringify(nl));
    const nlo = wheel.newLeafOpenersPublished || {};
    check('wd.those_leaves_name_a_published_owner',
      nlo.theme === true && nlo.religion === true && nlo.bridge === true && nlo.faction === true,
      JSON.stringify(nlo));

    // ── 2026-09-15 (g) THE CRIMINALITY HUB + THE CYBER-INTERCEPT DOOR ─────────────────────────
    // The hub is the leaf's real target — the leaf itself used to no-op for a clean player — and it
    // must OFFER every section it owns rather than merely existing.
    const cybSections = (wheel.criminalitySections || []).join(' | ');
    check('criminality.hub_opens_with_every_section',
      wheel.criminalityHub === true &&
      cybSections.indexOf('ARENA COURTHOUSE') >= 0 &&
      cybSections.indexOf('BOUNTY BOARD') >= 0 &&
      cybSections.indexOf('CYBER-INTERCEPT') >= 0,
      'hub=' + wheel.criminalityHub + ' sections=' + JSON.stringify(wheel.criminalitySections || []));
    // HONEST UI: a clean player owes nothing, so the courthouse control is DISABLED instead of opening
    // a courthouse that returns early. The assertion is a CONSISTENCY one, so it holds either way.
    check('criminality.courthouse_control_matches_the_wanted_level',
      (Number(wheel.criminalityWanted) > 0 && wheel.criminalityCourthouseDisabled === false) ||
      (!(Number(wheel.criminalityWanted) > 0) && wheel.criminalityCourthouseDisabled === true),
      'wanted=' + wheel.criminalityWanted + ' disabled=' + wheel.criminalityCourthouseDisabled);
    // The Intel-Agent door: the overlay opens AND the action its own markup names is published
    // (an unpublished handler is the dead-control class this probe exists to catch).
    check('criminality.cyber_intercept_is_reachable_and_its_action_published',
      wheel.cyberInterceptOverlay === true && wheel.cyberInterceptPublished === true,
      'overlay=' + wheel.cyberInterceptOverlay + ' actionPublished=' + wheel.cyberInterceptPublished +
      ' err=' + wheel.criminalityError);

    console.log('  DIAG  ' + JSON.stringify(ro.result.value.diag));
    console.log('-'.repeat(60));
    console.log(`[PROBE] ${pass} passed, ${fail} failed`);
    process.exitCode = fail ? 1 : 0;
  } catch (e) {
    console.error('[PROBE] error:', e.message);
    process.exitCode = 1;
  } finally {
    try { if (cdp) cdp.ws.close(); } catch (_) {}
    try { proc.kill(); } catch (_) {}
    try { fs.rmSync(profile, { recursive: true, force: true }); } catch (_) {}
  }
}

main();

