#!/usr/bin/env node
// ============================================================================
// Architecture Scanner — Autonomous Flow Detection
// ----------------------------------------------------------------------------
// Scans the codebase to detect:
//   1. Go WASM bridge functions (js.Global().Set)
//   2. JS ES module imports/exports
//   3. HTML standalone <script> tags
//   4. CSS SCSS partials
//   5. Build tag violations
//   6. Flow integrity (registration + handler pairing)
//
// Outputs: tools/archscan/manifest.json
// Usage:  node tools/archscan/scan.js
// ============================================================================

const fs = require('fs');
const path = require('path');

const ROOT = path.resolve(__dirname, '../..');
const PUBLIC = path.join(ROOT, 'Public');
const JS_DIR = path.join(PUBLIC, 'js');
const SCSS_DIR = path.join(PUBLIC, 'src', 'scss');
const GO_DIR = ROOT;
const MANIFEST_PATH = path.join(__dirname, 'manifest.json');

// ============================================================================
// UTILITIES
// ============================================================================

function readJSON(p) {
  try { return JSON.parse(fs.readFileSync(p, 'utf8')); } catch { return null; }
}

function writeJSON(p, data) {
  fs.writeFileSync(p, JSON.stringify(data, null, 2));
}

function glob(dir, ext) {
  if (!fs.existsSync(dir)) return [];
  return fs.readdirSync(dir)
    .filter(f => f.endsWith(ext))
    .map(f => path.join(dir, f));
}

function globRecursive(dir, ext) {
  if (!fs.existsSync(dir)) return [];
  const results = [];
  function walk(d) {
    for (const entry of fs.readdirSync(d, { withFileTypes: true })) {
      const full = path.join(d, entry.name);
      if (entry.isDirectory()) walk(full);
      else if (entry.name.endsWith(ext)) results.push(full);
    }
  }
  walk(dir);
  return results;
}

function lineAt(content, index) {
  return content.substring(0, index).split('\n').length;
}

// ============================================================================
// SCANNER: Go Bridge Functions (with flow integrity)
// ============================================================================

function scanGoBridge() {
  const registrations = new Map(); // name -> {file, line}
  const handlers = new Map();      // name -> {file, line}

  const files = glob(GO_DIR, '.go');

  for (const file of files) {
    const content = fs.readFileSync(file, 'utf8');
    const relPath = path.relative(ROOT, file);

    // Match js.Global().Set("FuncName", ...) — the registration
    const setPattern = /js\.Global\(\)\.Set\(\s*"(\w+)"/g;
    let match;
    while ((match = setPattern.exec(content)) !== null) {
      registrations.set(match[1], {
        file: relPath,
        line: lineAt(content, match.index)
      });
    }

    // Match func Xxx(this js.Value...) — the handler implementation
    const funcPattern = /func\s+(Get|Set|Sync|Handle|Toggle)\w*\s*\(this\s+js\.Value/g;
    while ((match = funcPattern.exec(content)) !== null) {
      const nameMatch = match[0].match(/func\s+(\w+)/);
      if (nameMatch) {
        handlers.set(nameMatch[1], {
          file: relPath,
          line: lineAt(content, match.index)
        });
      }
    }
  }

  // Merge into unified bridge functions with flow integrity
  const allNames = new Set([...registrations.keys(), ...handlers.keys()]);
  const functions = [];
  let completeFlows = 0;
  let registrationOnly = 0;
  let handlerOnly = 0;

  for (const name of allNames) {
    const reg = registrations.get(name);
    const hnd = handlers.get(name);
    const isComplete = reg && hnd;
    if (isComplete) completeFlows++;
    else if (reg) registrationOnly++;
    else handlerOnly++;

    functions.push({
      name,
      file: reg?.file || hnd?.file,
      registeredIn: reg?.file || null,
      implementedIn: hnd?.file || null,
      type: isComplete ? 'complete' : (reg ? 'registration-only' : 'handler-only'),
      line: reg?.line || hnd?.line,
      hasRegistration: !!reg,
      hasHandler: !!hnd
    });
  }

  const totalFlows = allNames.size;
  return {
    functions,
    flowIntegrity: {
      completeFlows,
      registrationOnly,
      handlerOnly,
      totalFlows,
      integrityPercent: totalFlows > 0 ? Math.round((completeFlows / totalFlows) * 100) : 100
    }
  };
}

// ============================================================================
// SCANNER: JS ES Modules
// ============================================================================

function scanJSModules() {
  const modules = [];
  const files = glob(JS_DIR, '.js');

  for (const file of files) {
    const content = fs.readFileSync(file, 'utf8');
    const relPath = path.relative(ROOT, file);
    const fileName = path.basename(file);

    // Detect IIFE pattern
    const hasIIFE = /^\s*\(function\s*\(\)/.test(content) || /^\s*\(\s*function/.test(content);

    // Detect ES module exports
    const exports = [];
    const exportPattern = /export\s+(?:function|const|let|class)\s+(\w+)/g;
    let match;
    while ((match = exportPattern.exec(content)) !== null) {
      exports.push(match[1]);
    }

    // Detect ES module imports
    const imports = [];
    const importPattern = /import\s+(?:{[^}]+}|[\w*]+)\s+from\s+['"]([^'"]+)['"]/g;
    while ((match = importPattern.exec(content)) !== null) {
      imports.push(match[1]);
    }

    // Detect window.X = assignments (bridge bindings)
    const windowBindings = [];
    const windowPattern = /window\.(\w+)\s*=/g;
    while ((match = windowPattern.exec(content)) !== null) {
      windowBindings.push(match[1]);
    }

    // Derive category from filename prefix (data-driven, not hardcoded)
    const category = deriveModuleCategory(fileName, relPath);

    modules.push({
      file: relPath,
      fileName,
      category,
      isIIFE: hasIIFE,
      isESModule: exports.length > 0 || imports.length > 0,
      exports,
      imports,
      windowBindings,
      exportCount: exports.length,
      importCount: imports.length,
      connectionCount: exports.length + imports.length + windowBindings.length
    });
  }

  return modules;
}

/**
 * Derive module category from filename and path.
 * Uses prefix patterns — add new prefixes here as the codebase grows.
 * This is the single source of truth for module categorization.
 */
function deriveModuleCategory(fileName, filePath) {
  const lower = fileName.toLowerCase();

  // Prefix-based categorization (order matters — first match wins)
  const prefixRules = [
    { prefix: ['app', 'config', 'utils'], category: 'core' },
    { prefix: ['game', 'deck', 'network', 'ui', 'audio', 'particle'], category: 'game' },
    { prefix: ['economy', 'wallet', 'admin', 'leaderboard', 'shop'], category: 'economy' },
    { prefix: ['criminality', 'rivalry', 'portfolio', 'social'], category: 'social' },
    { prefix: ['menu', 'controller', 'user_preference', 'pathway'], category: 'menu' },
    { prefix: ['collective', 'ai', 'citizen'], category: 'ai' }
  ];

  for (const rule of prefixRules) {
    if (rule.prefix.some(p => lower.startsWith(p))) {
      return rule.category;
    }
  }

  // Fallback: use parent directory if nested
  const dir = path.dirname(filePath);
  if (dir && dir !== '.' && dir !== 'Public/js') {
    return path.basename(dir);
  }

  return 'other';
}

// ============================================================================
// SCANNER: HTML Script Tags
// ============================================================================

function scanHTMLScripts() {
  const scripts = [];
  const htmlFiles = [path.join(PUBLIC, 'index.html')];

  for (const file of htmlFiles) {
    if (!fs.existsSync(file)) continue;
    const content = fs.readFileSync(file, 'utf8');
    const relPath = path.relative(ROOT, file);

    // Match <script src="..."> tags
    const scriptPattern = /<script\s+[^>]*src\s*=\s*["']([^"']+)["'][^>]*>/g;
    let match;
    while ((match = scriptPattern.exec(content)) !== null) {
      scripts.push({
        file: relPath,
        src: match[1],
        type: 'standalone',
        line: lineAt(content, match.index)
      });
    }

    // Match <script type="module" src="app.js">
    const modulePattern = /<script\s[^>]*type\s*=\s*["']module["'][^>]*src\s*=\s*["']([^"']+)["']/g;
    while ((match = modulePattern.exec(content)) !== null) {
      scripts.push({
        file: relPath,
        src: match[1],
        type: 'module',
        line: lineAt(content, match.index)
      });
    }
  }

  return scripts;
}

// ============================================================================
// SCANNER: CSS SCSS Partials
// ============================================================================

function scanSCSSPartials() {
  const partials = [];
  const files = globRecursive(SCSS_DIR, '.scss');

  for (const file of files) {
    const content = fs.readFileSync(file, 'utf8');
    const relPath = path.relative(ROOT, file);
    const fileName = path.basename(file);

    // Count rules (rough)
    const ruleCount = (content.match(/[.#][\w-]+\s*\{/g) || []).length;

    partials.push({
      file: relPath,
      fileName,
      isPartial: fileName.startsWith('_'),
      ruleCount
    });
  }

  return partials;
}

// ============================================================================
// SCANNER: Build Tag Violations
// ============================================================================

function scanBuildTags() {
  const violations = [];
  const goFiles = glob(GO_DIR, '.go');

  for (const file of goFiles) {
    const content = fs.readFileSync(file, 'utf8');
    const relPath = path.relative(ROOT, file);
    const lines = content.split('\n');

    for (let i = 0; i < lines.length; i++) {
      const line = lines[i];
      if (line.includes('//go:build') || line.includes('// +build')) {
        // Check if build tag is valid
        if (!line.match(/^\/\/\s*(go:build|\+build)/)) {
          violations.push({
            file: relPath,
            line: i + 1,
            content: line.trim(),
            issue: 'Malformed build tag'
          });
        }
      }
    }
  }

  return violations;
}

// ============================================================================
// CHANGE DETECTION
// ============================================================================

function detectChanges(oldManifest, newManifest) {
  const changes = {
    timestamp: new Date().toISOString(),
    added: [],
    removed: [],
    modified: []
  };

  if (!oldManifest) {
    changes.added.push({
      type: 'initial',
      message: 'Initial scan — no previous manifest found'
    });
    return changes;
  }

  // Compare bridge functions
  const oldBridge = new Set(oldManifest.bridgeFunctions?.map(f => f.name) || []);
  const newBridge = new Set(newManifest.bridgeFunctions?.map(f => f.name) || []);

  for (const name of newBridge) {
    if (!oldBridge.has(name)) {
      changes.added.push({ type: 'bridge', name });
    }
  }
  for (const name of oldBridge) {
    if (!newBridge.has(name)) {
      changes.removed.push({ type: 'bridge', name });
    }
  }

  // Compare JS modules
  const oldModules = new Map(oldManifest.jsModules?.map(m => [m.file, m]) || []);
  const newModules = new Map(newManifest.jsModules?.map(m => [m.file, m]) || []);

  for (const [file, mod] of newModules) {
    if (!oldModules.has(file)) {
      changes.added.push({ type: 'jsModule', file });
    } else {
      const old = oldModules.get(file);
      if (old.exportCount !== mod.exportCount || old.importCount !== mod.importCount) {
        changes.modified.push({
          type: 'jsModule',
          file,
          oldExports: old.exportCount,
          newExports: mod.exportCount,
          oldImports: old.importCount,
          newImports: mod.importCount
        });
      }
    }
  }
  for (const file of oldModules.keys()) {
    if (!newModules.has(file)) {
      changes.removed.push({ type: 'jsModule', file });
    }
  }

  // Compare HTML scripts
  const oldScripts = new Set(oldManifest.htmlScripts?.map(s => s.src) || []);
  const newScripts = new Set(newManifest.htmlScripts?.map(s => s.src) || []);

  for (const src of newScripts) {
    if (!oldScripts.has(src)) {
      changes.added.push({ type: 'script', src });
    }
  }
  for (const src of oldScripts) {
    if (!newScripts.has(src)) {
      changes.removed.push({ type: 'script', src });
    }
  }

  return changes;
}

// ============================================================================
// MAIN SCAN
// ============================================================================

function scan() {
  console.log('🔍 Scanning architecture...\n');

  const oldManifest = readJSON(MANIFEST_PATH);

  const { functions: bridgeFunctions, flowIntegrity } = scanGoBridge();

  const manifest = {
    timestamp: new Date().toISOString(),
    bridgeFunctions,
    flowIntegrity,
    jsModules: scanJSModules(),
    htmlScripts: scanHTMLScripts(),
    scssPartials: scanSCSSPartials(),
    buildTagViolations: scanBuildTags(),
    summary: {}
  };

  // Summary
  manifest.summary = {
    bridgeFunctions: manifest.bridgeFunctions.length,
    completeFlows: flowIntegrity.completeFlows,
    registrationOnly: flowIntegrity.registrationOnly,
    handlerOnly: flowIntegrity.handlerOnly,
    flowIntegrityPercent: flowIntegrity.integrityPercent,
    jsModules: manifest.jsModules.length,
    esModules: manifest.jsModules.filter(m => m.isESModule).length,
    iifeModules: manifest.jsModules.filter(m => m.isIIFE).length,
    htmlScripts: manifest.htmlScripts.length,
    scssPartials: manifest.scssPartials.length,
    violations: manifest.buildTagViolations.length
  };

  // Detect changes
  const changes = detectChanges(oldManifest, manifest);
  manifest.changes = changes;

  // Save manifest
  writeJSON(MANIFEST_PATH, manifest);

  // Output
  console.log('📊 Summary:');
  console.log(`   Bridge Functions: ${manifest.summary.bridgeFunctions} (${flowIntegrity.completeFlows} complete flows, ${flowIntegrity.integrityPercent}% integrity)`);
  console.log(`   JS Modules: ${manifest.summary.jsModules} (${manifest.summary.esModules} ES, ${manifest.summary.iifeModules} IIFE)`);
  console.log(`   HTML Scripts: ${manifest.summary.htmlScripts}`);
  console.log(`   SCSS Partials: ${manifest.summary.scssPartials}`);
  console.log(`   Build Tag Violations: ${manifest.summary.violations}`);

  if (changes.added.length > 0) {
    console.log('\n✅ Added:');
    changes.added.forEach(c => console.log(`   + ${c.type}: ${c.name || c.file || c.src || c.message}`));
  }
  if (changes.removed.length > 0) {
    console.log('\n❌ Removed:');
    changes.removed.forEach(c => console.log(`   - ${c.type}: ${c.name || c.file || c.src}`));
  }
  if (changes.modified.length > 0) {
    console.log('\n🔄 Modified:');
    changes.modified.forEach(c => console.log(`   ~ ${c.file}: exports ${c.oldExports}→${c.newExports}, imports ${c.oldImports}→${c.newImports}`));
  }

  console.log(`\n💾 Manifest saved to: ${path.relative(ROOT, MANIFEST_PATH)}`);

  return manifest;
}

// ============================================================================
// RUN
// ============================================================================

if (require.main === module) {
  scan();
}

module.exports = { scan, scanGoBridge, scanJSModules, scanHTMLScripts, scanSCSSPartials, deriveModuleCategory };
