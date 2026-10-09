#!/usr/bin/env node
// ============================================================================
// Architecture Watcher — Continuous Flow Detection
// ----------------------------------------------------------------------------
// Watches the codebase for changes and updates BOTH manifests automatically.
//   - manifest.json      → architecture bridge functions, JS modules, HTML scripts
//   - wired-manifest.json → wired vs orphaned file detection
//
// Usage: node tools/archscan/watch.js
// ============================================================================

const fs = require('fs');
const path = require('path');
const { scan } = require('./scan.js');
const { detectWiredOrphaned } = require('./wired.js');

const ROOT = path.resolve(__dirname, '../..');
const ARCH_PATH = path.join(__dirname, 'manifest.json');
const WIRED_PATH = path.join(__dirname, 'wired-manifest.json');
const WATCH_INTERVAL = 5000;

let isScanning = false;
let lastArchManifest = null;
let lastWiredManifest = null;

function loadArchManifest() {
  try { return JSON.parse(fs.readFileSync(ARCH_PATH, 'utf8')); } catch { return null; }
}

function loadWiredManifest() {
  try { return JSON.parse(fs.readFileSync(WIRED_PATH, 'utf8')); } catch { return null; }
}

function watch() {
  console.log('👁️  Architecture Watcher started');
  console.log('   Watching: ' + ROOT);
  console.log('   Interval: ' + WATCH_INTERVAL + 'ms');
  console.log('   Watching: manifest.json + wired-manifest.json');
  console.log('   Press Ctrl+C to stop\n');

  // Initial scan
  lastArchManifest = scan();
  lastWiredManifest = detectWiredOrphaned();

  setInterval(() => {
    if (isScanning) return;
    isScanning = true;

    try {
      // Refresh architecture manifest
      const newArchManifest = scan();
      const archChanges = detectArchChanges(lastArchManifest, newArchManifest);
      
      // Refresh wired manifest
      const newWiredManifest = detectWiredOrphaned();
      const wiredChanges = detectWiredChanges(lastWiredManifest, newWiredManifest);
      
      // Report changes
      if (archChanges || wiredChanges) {
        console.log('\n🔔 Changes detected!');
        if (archChanges) console.log('  📐 Architecture: ' + archChanges);
        if (wiredChanges) console.log('  🔌 Wired: ' + wiredChanges);
        console.log();
        lastArchManifest = newArchManifest;
        lastWiredManifest = newWiredManifest;
      }
    } catch (e) {
      console.error('Scan error:', e.message);
    } finally {
      isScanning = false;
    }
  }, WATCH_INTERVAL);
}

function detectArchChanges(old, neu) {
  if (!old || !neu) return null;
  const oldBridge = new Set(old.bridgeFunctions?.map(f => f.name) || []);
  const newBridge = new Set(neu.bridgeFunctions?.map(f => f.name) || []);
  const added = [...newBridge].filter(x => !oldBridge.has(x));
  const removed = [...oldBridge].filter(x => !newBridge.has(x));
  
  if (added.length || removed.length) {
    const parts = [];
    if (added.length) parts.push('+' + added.length + ' bridge funcs');
    if (removed.length) parts.push('-' + removed.length + ' bridge funcs');
    return parts.join(', ');
  }
  return null;
}

function detectWiredChanges(old, neu) {
  if (!old || !neu) return null;
  const oldWired = old.stats?.wired || 0;
  const newWired = neu.stats?.wired || 0;
  const diff = newWired - oldWired;
  
  if (diff !== 0) {
    return (diff > 0 ? '+' : '') + diff + ' wired files (now ' + newWired + '/' + neu.stats.total + ')';
  }
  return null;
}

// Run
if (require.main === module) {
  watch();
}

module.exports = { watch };
