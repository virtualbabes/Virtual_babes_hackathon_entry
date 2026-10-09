#!/usr/bin/env node
// ============================================================================
// Architecture Renderer v5 — Template-Driven
// ----------------------------------------------------------------------------
// Reads manifests, computes analytics (including proper health score),
// reads the HTML template, injects data, and writes the diagram.
//
// Usage: node tools/archscan/render.js
// Output: AI-Brain/architecture.html
// ============================================================================

const fs = require('fs');
const path = require('path');

const ROOT = path.resolve(__dirname, '../..');
const MANIFEST_PATH = path.join(__dirname, 'manifest.json');
const WIRED_PATH = path.join(__dirname, 'wired-manifest.json');
const TEMPLATE_PATH = path.join(__dirname, 'template.html');
const OUTPUT_PATH = path.join(ROOT, 'AI-Brain', 'architecture.html');
const MASTER_OUTPUT_PATH = path.join(__dirname, 'architecture.html');

// ============================================================================
// READ MANIFESTS
// ============================================================================

function readManifest() {
  try { return JSON.parse(fs.readFileSync(MANIFEST_PATH, 'utf8')); } catch { return require('./scan.js').scan(); }
}

function readWired() {
  try { return JSON.parse(fs.readFileSync(WIRED_PATH, 'utf8')); } catch { return require('./wired.js').detectWiredOrphaned(); }
}

// ============================================================================
// ANALYTICS CALCULATION
// ============================================================================

function computeAnalytics(manifest, wired) {
  const { bridgeFunctions, jsModules, scssPartials, summary } = manifest;
  const flowIntegrity = manifest.flowIntegrity || {
    completeFlows: summary.bridgeFunctions,
    registrationOnly: 0,
    handlerOnly: 0,
    totalFlows: summary.bridgeFunctions,
    integrityPercent: 100
  };

  const totalExports = jsModules.reduce((sum, m) => sum + m.exportCount, 0);
  const totalImports = jsModules.reduce((sum, m) => sum + m.importCount, 0);
  const totalWindowBindings = jsModules.reduce((sum, m) => sum + m.windowBindings.length, 0);

  // Language distribution — counted from the actual repo inventory, not bridge guesses
  const inventory = [...(wired.wired || []), ...(wired.orphaned || [])];
  const langCounts = {};
  const LANG_RULES = [
    [/\.go$/i, 'Go'],
    [/\.(js|mjs|cjs|jsx)$/i, 'JavaScript'],
    [/\.(ts|tsx)$/i, 'TypeScript'],
    [/\.(css|scss|sass|less)$/i, 'CSS'],
    [/\.html?$/i, 'HTML'],
    [/\.md$/i, 'Markdown'],
    [/\.json$/i, 'JSON'],
    [/\.py$/i, 'Python'],
    [/\.(ps1|sh|bash|bat|cmd)$/i, 'Shell'],
    [/\.(png|jpe?g|gif|svg|ico|webp|bmp)$/i, 'Images'],
    [/\.(mp3|wav|ogg|mp4|webm|woff2?|ttf|otf|eot)$/i, 'Media']
  ];
  for (const f of inventory) {
    const p = f.path || '';
    let label = 'Other';
    for (const [rx, name] of LANG_RULES) { if (rx.test(p)) { label = name; break; } }
    if (!langCounts[label]) langCounts[label] = { label, files: 0, bytes: 0 };
    langCounts[label].files++;
    langCounts[label].bytes += f.size || 0;
  }
  const languages = Object.values(langCounts).sort((a, b) => b.files - a.files);
  const goFiles = (langCounts.Go && langCounts.Go.files) || 0;
  const jsFiles = ((langCounts.JavaScript && langCounts.JavaScript.files) || 0)
    + ((langCounts.TypeScript && langCounts.TypeScript.files) || 0);
  const cssFiles = (langCounts.CSS && langCounts.CSS.files) || 0;
  const totalFiles = inventory.length || wired.stats.total;

  // Module connectedness — modules with at least 1 import/export
  const connectedModules = jsModules.filter(m => m.connectionCount > 0).length;

  // ===== HEALTH SCORE CALCULATION =====
  // 4 factors, each contributing up to 25 points (100 total):

  // Factor 1: Wired ratio (0-25 points)
  // High wired ratio = good (most files are connected to the architecture)
  const wiredRatio = wired.stats.wired / Math.max(wired.stats.total, 1);
  const wiredScore = Math.round(wiredRatio * 25);

  // Factor 2: Flow integrity (0-25 points)
  // Complete flows (registration + handler) / total bridge functions
  const integrityRatio = flowIntegrity.integrityPercent / 100;
  const integrityScore = Math.round(integrityRatio * 25);

  // Factor 3: Module connectedness (0-25 points)
  // ES modules with at least 1 import or export / total JS modules
  const connectednessRatio = connectedModules / Math.max(jsModules.length, 1);
  const connectednessScore = Math.round(connectednessRatio * 25);

  // Factor 4: Build tag violations (0-25 points, inverted)
  // No violations = full points, each violation costs 5 points
  const violationPenalty = Math.min(25, (summary.violations || 0) * 5);
  const violationScore = 25 - violationPenalty;

  const healthScore = wiredScore + integrityScore + connectednessScore + violationScore;

  return {
    summary,
    flowIntegrity,
    totalExports,
    totalImports,
    totalWindowBindings,
    totalFiles,
    goFiles,
    jsFiles,
    cssFiles,
    languages,
    connectedModules,
    healthScore,
    wiredScore,
    integrityScore,
    connectednessScore,
    violationScore,
    wiredRatio,
    integrityRatio,
    connectednessRatio,
    violationCount: summary.violations || 0
  };
}

// ============================================================================
// HTML TEMPLATE INJECTION
// ============================================================================

/**
 * Safely embed JSON into a <script> tag.
 * Prevents `</script>` from breaking out of the script context.
 */
function safeJSON(obj) {
  return JSON.stringify(obj).replace(/<\//g, '<\\/');
}

/**
 * Read the view definitions from views.js for injection into the template.
 * This keeps the JS logic in a separate file while still embedding it inline.
 */
function getViewDefinitions() {
  const viewsContent = fs.readFileSync(path.join(__dirname, 'views.js'), 'utf8');

  // Extract everything from the first function definition to the last before exports
  // We want: constants + all view/chart functions + helper functions
  const lines = viewsContent.split('\n');

  // Find the start (first const or function)
  let startIdx = 0;
  for (let i = 0; i < lines.length; i++) {
    const trimmed = lines[i].trim();
    if (trimmed.startsWith('const ') || trimmed.startsWith('function ')) {
      startIdx = i;
      break;
    }
  }

  // Find the exports section (we don't want to inject module.exports)
  let endIdx = lines.length;
  for (let i = lines.length - 1; i >= 0; i--) {
    if (lines[i].trim().startsWith('module.exports')) {
      endIdx = i;
      break;
    }
  }

  return lines.slice(startIdx, endIdx).join('\n');
}

function generateHTML(manifest, wired, analytics) {
  let template = fs.readFileSync(TEMPLATE_PATH, 'utf8');

  // Inject data
  template = template.replace('__MANIFEST_DATA__', safeJSON(manifest));
  template = template.replace('__WIRED_DATA__', safeJSON(wired));
  template = template.replace('__ANALYTICS_DATA__', safeJSON(analytics));

  // Inject view definitions
  template = template.replace('__VIEW_DEFINITIONS__', getViewDefinitions());

  return template;
}

// ============================================================================
// MAIN
// ============================================================================

function main() {
  console.log('🎨 Rendering architecture diagram v5...\n');

  const manifest = readManifest();
  const wired = readWired();
  const analytics = computeAnalytics(manifest, wired);

  const html = generateHTML(manifest, wired, analytics);

  // Ensure output directory exists
  const outputDir = path.dirname(OUTPUT_PATH);
  if (!fs.existsSync(outputDir)) {
    fs.mkdirSync(outputDir, { recursive: true });
  }

  fs.writeFileSync(OUTPUT_PATH, html);
  fs.writeFileSync(MASTER_OUTPUT_PATH, html);

  console.log(`✅ Diagram updated: ${path.relative(ROOT, OUTPUT_PATH)}`);
  console.log(`✅ Diagram updated (master): ${path.relative(ROOT, MASTER_OUTPUT_PATH)}`);
  console.log(`   Bridge Functions: ${manifest.summary.bridgeFunctions} (${manifest.flowIntegrity?.integrityPercent || 0}% integrity)`);
  console.log(`   ES Modules: ${manifest.summary.esModules}`);
  console.log(`   Health: ${analytics.healthScore}% (wired: ${analytics.wiredScore}, flow: ${analytics.integrityScore}, connected: ${analytics.connectednessScore}, violations: ${analytics.violationScore})`);
  console.log(`   Wired: ${wired.stats.wired}/${wired.stats.total} (${wired.stats.wiredPercent}%)`);
  console.log(`   Orphaned: ${wired.stats.orphaned}`);
}

if (require.main === module) {
  main();
}

module.exports = { generateHTML, computeAnalytics, main };
