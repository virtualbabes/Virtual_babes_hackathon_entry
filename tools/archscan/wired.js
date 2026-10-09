#!/usr/bin/env node
// ============================================================================
// Wired vs Orphaned Detector — reference-graph driven
// ----------------------------------------------------------------------------
// WIRED  = file reachable from an entry point (HTML script, JS import,
//          SCSS @import, Go build unit with a func main(), or core infra)
// ORPHAN = file that exists but nothing references it
//
// Usage:  node tools/archscan/wired.js
// Output: tools/archscan/wired-manifest.json
// ============================================================================

const fs = require('fs');
const path = require('path');
const { ROOT, scanRepoTree, buildReferenceGraph } = require('./archlib.js');

const OUTPUT_PATH = path.join(__dirname, 'wired-manifest.json');

// ----------------------------------------------------------------------------
// Classification
// ----------------------------------------------------------------------------

const CATEGORY_RULES = [
  ['media',     /\.(png|jpe?g|gif|svg|ico|mp3|wav|ogg|mp4|webm|woff2?|ttf|otf|eot)$/i],
  ['generated', /\.wasm$|node_modules\//i],
  ['docs',      /\.md$/i],
  ['config',    /\.(json|ya?ml|toml|ini|cfg|conf|env|gitignore|editorconfig)$/i],
  ['backend',   /\.go$/i],
  ['frontend',  /\.(js|ts|jsx|tsx)$/i],
  ['styles',    /\.(css|scss|sass|less)$/i]
];

function classify(relPath) {
  if (relPath.includes('vendor/')) return 'vendor';
  for (const [cat, rx] of CATEGORY_RULES) if (rx.test(relPath)) return cat;
  return 'code';
}

// pass-through categories: wired by nature (never flagged as orphans)
const PASS_THROUGH = new Set(['vendor', 'generated', 'docs', 'config']);

// ----------------------------------------------------------------------------
// Main detection
// ----------------------------------------------------------------------------

function detectWiredOrphaned() {
  console.log('🔌 Detecting wired vs orphaned files...\n');

  const repoFiles = scanRepoTree();
  const { referenced } = buildReferenceGraph();

  const result = {
    timestamp: new Date().toISOString(),
    wired: [],
    orphaned: [],
    stats: { total: repoFiles.length, wired: 0, orphaned: 0, wiredPercent: 0 },
    byCategory: {}
  };

  for (const file of repoFiles) {
    const category = classify(file.path);
    const isReferenced = referenced.has(file.path);
    const wired = isReferenced || PASS_THROUGH.has(category);

    const entry = { path: file.path, category, size: file.size, isReferenced };

    if (wired) { result.wired.push(entry); result.stats.wired++; }
    else       { result.orphaned.push(entry); result.stats.orphaned++; }

    if (!result.byCategory[category]) result.byCategory[category] = { wired: 0, orphaned: 0 };
    result.byCategory[category][wired ? 'wired' : 'orphaned']++;
  }

  result.stats.wiredPercent = result.stats.total
    ? Math.round((result.stats.wired / result.stats.total) * 100) : 0;

  fs.writeFileSync(OUTPUT_PATH, JSON.stringify(result, null, 2));

  // ---- console summary ----
  console.log('📊 Wired vs Orphaned Summary:');
  console.log(`   Total Files: ${result.stats.total}`);
  console.log(`   Wired: ${result.stats.wired} (${result.stats.wiredPercent}%)`);
  console.log(`   Orphaned: ${result.stats.orphaned} (${100 - result.stats.wiredPercent}%)\n`);

  console.log('📁 By Category:');
  Object.entries(result.byCategory)
    .sort((a, b) => (b[1].wired + b[1].orphaned) - (a[1].wired + a[1].orphaned))
    .forEach(([cat, c]) => {
      const total = c.wired + c.orphaned;
      console.log(`   ${cat}: ${c.wired} wired, ${c.orphaned} orphaned (${Math.round((c.wired / total) * 100)}% wired)`);
    });

  if (result.orphaned.length) {
    console.log('\n🔴 Orphaned Files (sample):');
    result.orphaned.slice(0, 15).forEach(f => console.log(`   ${f.path} (${formatSize(f.size)})`));
    if (result.orphaned.length > 15) console.log(`   ... and ${result.orphaned.length - 15} more`);
  }

  console.log(`\n💾 Wired manifest saved to: ${path.relative(ROOT, OUTPUT_PATH)}`);
  return result;
}

function formatSize(bytes) {
  if (bytes < 1024) return bytes + ' B';
  if (bytes < 1024 * 1024) return (bytes / 1024).toFixed(1) + ' KB';
  return (bytes / (1024 * 1024)).toFixed(1) + ' MB';
}

if (require.main === module) detectWiredOrphaned();

module.exports = { detectWiredOrphaned, formatSize };
