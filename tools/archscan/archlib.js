#!/usr/bin/env node
// ============================================================================
// archscan architecture library — shared scanning + reference graph
// ----------------------------------------------------------------------------
// Single source of truth for:
//   - Gitignore-compliant repo tree walking (negation + anchored patterns)
//   - Cross-language reference graph (JS imports → SCSS @import → Go → HTML)
// Used by scan.js, wired.js, render.js, watch.js.
// ============================================================================

const fs = require('fs');
const path = require('path');

const ROOT = path.resolve(__dirname, '../..');

// ----------------------------------------------------------------------------
// Gitignore handling (full negation + anchored pattern support)
// ----------------------------------------------------------------------------

function loadGitignore() {
  const rules = [];
  let content = '';
  try { content = fs.readFileSync(path.join(ROOT, '.gitignore'), 'utf8'); } catch { /* absent */ }

  for (let line of content.split('\n')) {
    line = line.replace(/\r$/, '');
    if (!line || line.startsWith('#')) continue;
    let negate = false;
    if (line.startsWith('!')) { negate = true; line = line.slice(1); }
    if (!line) continue;
    const anchored = line.startsWith('/');
    if (anchored) line = line.slice(1);
    const dirOnly = line.endsWith('/');
    if (dirOnly) line = line.slice(0, -1);
    rules.push({ pattern: line, negate, anchored, dirOnly });
  }

  // scanner-internal ignores (always)
  const forced = ['.git', '.DS_Store', 'Thumbs.db', 'node_modules', 'tools'];
  forced.forEach(f => rules.push({ pattern: f, negate: false, anchored: false, dirOnly: false }));
  return rules;
}

function globToRegex(pattern) {
  const rx = pattern
    .replace(/[.+^${}()|[\]\\]/g, '\\$&')
    .replace(/\*\*/g, '')
    .replace(/\*/g, '[^/]*')
    .replace(//g, '.*')
    .replace(/\?/g, '[^/]');
  return new RegExp('^' + rx + '$');
}

function matchesRule(relPath, rule) {
  const regex = globToRegex(rule.pattern);
  if (rule.anchored) return regex.test(relPath);
  // unanchored: match any trailing path segment sequence
  const parts = relPath.split('/');
  for (let i = 0; i < parts.length; i++) {
    const sub = parts.slice(i).join('/');
    if (regex.test(sub)) return true;
  }
  // directory basename match (e.g. "node_modules")
  return parts.some(p => regex.test(p));
}

function isIgnored(relPath, rules) {
  let ignored = false;
  for (const rule of rules) {
    if (matchesRule(relPath, rule)) ignored = !rule.negate;
  }
  return ignored;
}

// ----------------------------------------------------------------------------
// Repository tree
// ----------------------------------------------------------------------------

function scanRepoTree() {
  const rules = loadGitignore();
  const files = [];
  (function walk(dir) {
    let entries;
    try { entries = fs.readdirSync(dir, { withFileTypes: true }); } catch { return; }
    for (const entry of entries) {
      const full = path.join(dir, entry.name);
      const rel = path.relative(ROOT, full).replace(/\\/g, '/');
      if (isIgnored(rel, rules)) continue;
      if (entry.isDirectory()) walk(full);
      else if (entry.isFile()) {
        const stat = fs.statSync(full);
        files.push({ path: rel, size: stat.size, modified: stat.mtime.toISOString() });
      }
    }
  })(ROOT);
  return files;
}

// ----------------------------------------------------------------------------
// Reference graph — resolves who references whom across languages
// ----------------------------------------------------------------------------

const REMOVED = null;

function resolveRelative(fromFile, spec, baseDir) {
  // spec like './x.js' or '../shared/y'
  const base = path.posix.dirname(baseDir);
  let resolved = path.posix.normalize(path.posix.join(base, spec));
  const candidates = [resolved, resolved + '.js', resolved + '/index.js'];
  if (resolved.endsWith('.js')) candidates.unshift(resolved);
  for (const c of candidates) if (fs.existsSync(path.join(ROOT, c))) return c;
  return null;
}

function resolveScssImport(scssRoot, fromFile, spec) {
  // spec: 'base/variables' or 'components/buttons'
  const fromDir = path.posix.dirname(fromFile);
  const tries = [];
  // relative to importing file
  tries.push(path.posix.normalize(path.posix.join(fromDir, spec)));
  // relative to scss root (sass convention)
  tries.push(path.posix.normalize(path.posix.join(scssRoot, spec)));
  const variants = [];
  for (const t of tries) {
    const dir = path.posix.dirname(t), name = path.posix.basename(t);
    variants.push(t, t + '.scss', path.posix.join(dir, '_' + name + '.scss'));
  }
  for (const v of variants) if (fs.existsSync(path.join(ROOT, v))) return v;
  return null;
}

function buildReferenceGraph() {
  const referenced = new Set();
  const edges = []; // { from, to, kind } for flow health visualization

  const addRef = (from, to, kind) => {
    if (!to) return;
    referenced.add(to);
    edges.push({ from, to, kind });
  };

  const read = p => { try { return fs.readFileSync(path.join(ROOT, p), 'utf8'); } catch { return ''; } };

  // --- HTML ---
  const htmlFiles = ['Public/index.html'].filter(f => fs.existsSync(path.join(ROOT, f)));
  for (const html of htmlFiles) {
    referenced.add(html);
    const content = read(html);
    const re = /<script\s[^>]*src\s*=\s*["']([^"']+)["']/g;
    let m;
    while ((m = re.exec(content)) !== null) {
      let src = m[1];
      if (/^https?:/.test(src)) continue;                      // CDN scripts are external, not repo files
      src = src.replace(/^\//, '');
      const resolved = fs.existsSync(path.join(ROOT, src)) ? src
        : fs.existsSync(path.join(ROOT, 'Public', src)) ? path.posix.join('Public', src) : null;
      if (resolved) addRef(html, resolved, 'html-script');
    }
  }

  // --- JS modules (imports resolved relative to the importing file) ---
  const jsDir = 'Public/js';
  if (fs.existsSync(path.join(ROOT, jsDir))) {
    for (const f of scanRepoTree().filter(f => f.path.endsWith('.js')).map(f => f.path)) {
      const content = read(f);
      const importRe = /import\s+(?:[^'"]*?\s+from\s+)?['"]([^'"]+)['"]/g;
      let m;
      while ((m = importRe.exec(content)) !== null) {
        const spec = m[1];
        if (/^https?:/.test(spec) || spec.startsWith('/')) continue; // CDN/absolute vendor
        const resolved = resolveRelative(path.join(ROOT, f), spec, f);
        if (resolved) addRef(f, resolved, 'js-import');
      }
    }
  }

  // --- SCSS (@import/@use graph from entry point) ---
  const scssRoot = 'Public/src/scss';
  const scssEntry = path.posix.join(scssRoot, 'main.scss');
  if (fs.existsSync(path.join(ROOT, scssEntry))) {
    const queue = [scssEntry];
    const seen = new Set();
    while (queue.length) {
      const current = queue.pop();
      if (seen.has(current)) continue;
      seen.add(current);
      referenced.add(current);
      const content = read(current);
      const importRe = /@(?:import|use|forward)\s+['"]([^'"]+)['"]/g;
      let m;
      while ((m = importRe.exec(content)) !== null) {
        const resolved = resolveScssImport(scssRoot, current, m[1]);
        if (resolved) { addRef(current, resolved, 'scss-import'); queue.push(resolved); }
      }
    }
  }

  // --- Go: build-constraint set-membership wiring ---
  // A .go file may compile into multiple binaries (untagged + `!js && !wasm`
  // files join both the server AND console builds). A file is wired if ANY
  // build that includes it also contains a `func main()`.
  const goFiles = scanRepoTree().filter(f => f.path.endsWith('.go')).map(f => f.path);
  const tagOf = f => {
    const m = read(f).slice(0, 2000).match(/\/\/go:build\s+([^\n]+)/);
    return m ? m[1].trim().replace(/\r$/, '') : '';
  };
  // Build-constraint membership, per the split-WASM architecture:
  //   server:  //go:build !js && !wasm        (authoritative server binary)
  //   wasm:    //go:build js && wasm          (client sim, mirrors via app.js/wasm_exec.js)
  //   console: //go:build console             (loopback dev build)
  // A negated term (!x) EXCLUDES build x; a bare term includes only builds
  // satisfying it. Untagged files join every build.
  const NEG = t => { const re = new RegExp('!' + t + '\\b'); return s => re.test(s); };
  const POS = t => { const re = new RegExp('(?<![!\\w])' + t + '\\b'); return s => re.test(s); };
  const inBuild = (tag, build) => {
    if (/!ignore/.test(tag) || tag === 'ignore') return false;
    if (NEG('js')(tag) && build === 'wasm') return false;
    if (NEG('wasm')(tag) && build === 'wasm') return false;
    if (NEG('console')(tag) && build === 'console') return false;
    // positive js/wasm requirements restrict to wasm build
    if (POS('js')(tag) || POS('wasm')(tag)) return build === 'wasm';
    // positive console requirement restricts to console build
    if (POS('console')(tag)) return build === 'console';
    return true; // untagged or pure negations not excluding this build
  };
  const BUILDS = ['server', 'console', 'wasm'];
  const buildsWithMain = new Set();
  for (const f of goFiles) {
    const tag = tagOf(f);
    if (/\bfunc\s+main\s*\(/.test(read(f))) {
      for (const b of BUILDS) if (inBuild(tag, b)) buildsWithMain.add(b);
    }
  }
  for (const f of goFiles) {
    const tag = tagOf(f);
    if (BUILDS.some(b => buildsWithMain.has(b) && inBuild(tag, b))) referenced.add(f);
  }

  // --- Always-wired project infrastructure ---
  const alwaysWired = [
    'package.json', 'go.mod', 'go.sum', 'Dockerfile', '.gitignore',
    'Public/wasm_exec.js', scssEntry, 'Public/index.html'
  ];
  alwaysWired.forEach(f => { if (fs.existsSync(path.join(ROOT, f))) referenced.add(f); });

  return { referenced, edges, inBuild };
}

module.exports = {
  ROOT, loadGitignore, isIgnored, scanRepoTree,
  buildReferenceGraph, resolveRelative, resolveScssImport
};
