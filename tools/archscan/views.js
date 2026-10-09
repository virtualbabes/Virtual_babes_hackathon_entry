#!/usr/bin/env node
// ============================================================================
// Architecture View Definitions — Data-Driven
// ----------------------------------------------------------------------------
// Each view is a function(g, data) that renders into an SVG group.
// Categories are derived from directory/filename prefixes — no hardcoded lists.
// ============================================================================

/**
 * Derive a category from a JS module's file path.
 * Uses prefix patterns — extend this map as the codebase grows.
 */
const CATEGORY_PREFIXES = [
  { prefix: 'app', category: 'core' },
  { prefix: 'config', category: 'core' },
  { prefix: 'utils', category: 'core' },
  { prefix: 'game', category: 'game' },
  { prefix: 'deck', category: 'game' },
  { prefix: 'network', category: 'game' },
  { prefix: 'ui', category: 'game' },
  { prefix: 'audio', category: 'game' },
  { prefix: 'particle', category: 'game' },
  { prefix: 'economy', category: 'economy' },
  { prefix: 'wallet', category: 'economy' },
  { prefix: 'admin', category: 'economy' },
  { prefix: 'leaderboard', category: 'economy' },
  { prefix: 'shop', category: 'economy' },
  { prefix: 'criminality', category: 'social' },
  { prefix: 'rivalry', category: 'social' },
  { prefix: 'portfolio', category: 'social' },
  { prefix: 'social', category: 'social' },
  { prefix: 'menu', category: 'menu' },
  { prefix: 'controller', category: 'menu' },
  { prefix: 'user_preference', category: 'menu' },
  { prefix: 'pathway', category: 'menu' },
  { prefix: 'collective', category: 'ai' },
  { prefix: 'ai', category: 'ai' },
  { prefix: 'citizen', category: 'ai' }
];

function getCategory(fileName) {
  const lower = fileName.toLowerCase();
  for (const rule of CATEGORY_PREFIXES) {
    if (lower.startsWith(rule.prefix)) return rule.category;
  }
  return 'other';
}

/**
 * Group JS modules by their derived category.
 */
function groupModulesByCategory(jsModules) {
  const groups = {};
  jsModules.forEach(mod => {
    const cat = mod.category || getCategory(mod.fileName);
    if (!groups[cat]) groups[cat] = [];
    groups[cat].push(mod);
  });
  return groups;
}

/**
 * Group bridge functions by their prefix (Get*, Set*, Sync*, etc.)
 */
function groupBridgesByPrefix(bridgeFunctions) {
  const groups = {};
  bridgeFunctions.forEach(bf => {
    const match = bf.name.match(/^(Get|Set|Sync|Handle|Toggle|Place|Start|Add|Remove|Trigger|Force|Compute|Init|Play|Reset|Select|Import|Auto|Complete|Push|Resolve|Apply|Breed|Adopt|Progress|Host|Capture|Detect|Evaluate|Combine|Render|Update|Create|Deploy|Execute|Spread|Send|Accept|Dissolve|Release|Reclaim|Claim|Join|Register|Promote|Issue|Buy|Sell|Trade|Take|Repay|Divide|Stabilize|Freeze|Unfreeze|Vote|Delegate|Propose|Close|Record|Refresh|Switch|Filter|Fetch|Load|Save|Destroy|Enable|Disable|Show|Hide|Open|Build|Rebuild|Initiate|Validate|Verify|Check|Calculate|Format|Parse|Esc|Fmt|Career|GetActive|MyClub|SendChallenge|Challenge|OpenRivalry)/);
    const cat = match ? match[1] : 'Other';
    if (!groups[cat]) groups[cat] = [];
    groups[cat].push(bf);
  });
  return groups;
}

// ============================================================================
// VIEW: Full System
// ============================================================================
function viewFull(g, data) {
  
  svg.setAttribute('viewBox', '0 0 1100 700');

  const defs = document.createElementNS('http://www.w3.org/2000/svg', 'defs');
  defs.innerHTML = '<pattern id="grid" width="40" height="40" patternUnits="userSpaceOnUse"><path d="M 40 0 L 0 0 0 40" fill="none" stroke="#1e293b" stroke-width="0.5"/></pattern>';
  g.appendChild(defs);
  g.appendChild(makeRect(0, 0, 1100, 700, 'url(#grid)', 'none'));

  g.appendChild(makeText(550, 25, 'NFT-Seduction Full System Architecture', 12, 'white', '700'));

  // Layer 1: Client
  g.appendChild(makeRect(15, 45, 1070, 140, 'rgba(8,51,68,0.2)', '#22d3ee', 10));
  g.appendChild(makeText(30, 60, 'CLIENT (Browser)', 9, '#22d3ee', '700'));
  addNode(g, 30, 75, 90, 40, 'rgba(8,51,68,0.4)', '#22d3ee', 'index.html', 'Entry', 'HTML entry', ['app']);
  addNode(g, 140, 75, 90, 40, 'rgba(76,29,149,0.4)', '#a78bfa', 'wasm_exec.js', 'Runtime', 'Go WASM runtime', ['app']);
  addNode(g, 250, 75, 120, 40, 'rgba(8,51,68,0.4)', '#22d3ee', 'app.js', 'Orchestrator', 'Imports all modules', ['wasm', 'domain']);
  addNode(g, 390, 75, 90, 40, 'rgba(8,51,68,0.4)', '#22d3ee', 'Domain JS', 'ES Modules', 'All domain logic', ['wasm']);
  addNode(g, 500, 75, 90, 40, 'rgba(76,29,149,0.4)', '#a78bfa', 'main.wasm', 'State Machine', 'Authoritative state', []);
  addNode(g, 610, 75, 90, 40, 'rgba(120,53,15,0.3)', '#fbbf24', 'SCSS/CSS', 'Styling', '60+ clip-paths', []);
  g.appendChild(makeLine(120, 95, 138, 95, '#64748b'));
  g.appendChild(makeLine(230, 95, 248, 95, '#64748b'));
  g.appendChild(makeLine(370, 95, 388, 95, '#64748b'));
  g.appendChild(makeLine(480, 95, 498, 95, '#64748b'));
  g.appendChild(makeLine(590, 95, 608, 95, '#64748b'));

  // Layer 2: WASM Bridge
  g.appendChild(makeRect(15, 200, 1070, 120, 'rgba(76,29,149,0.15)', '#a78bfa', 10));
  g.appendChild(makeText(30, 215, 'WASM BRIDGE', 9, '#a78bfa', '700'));
  addNode(g, 30, 230, 100, 40, 'rgba(76,29,149,0.4)', '#a78bfa', 'bridge_service.go', 'Hook Registry', '50+ functions', ['getsync']);
  addNode(g, 150, 230, 100, 40, 'rgba(76,29,149,0.4)', '#a78bfa', 'GetGameState()', 'State Export', 'JSON by scope', []);
  addNode(g, 270, 230, 100, 40, 'rgba(76,29,149,0.4)', '#a78bfa', 'SyncX()', 'State Ingest', 'SyncFullProfile, etc', []);
  addNode(g, 390, 230, 100, 40, 'rgba(76,29,149,0.4)', '#a78bfa', 'menu_state.go', 'Menu Bridge', '8 functions', ['ctrl']);
  addNode(g, 510, 230, 100, 40, 'rgba(76,29,149,0.4)', '#a78bfa', 'ReplayEngine', 'Catch-up', '100 frame buffer', []);

  // Layer 3: Server
  g.appendChild(makeRect(15, 335, 1070, 150, 'rgba(6,78,59,0.15)', '#34d399', 10));
  g.appendChild(makeText(30, 350, 'SERVER (Go)', 9, '#34d399', '700'));
  addNode(g, 30, 365, 100, 40, 'rgba(6,78,59,0.4)', '#34d399', 'server.go', 'HTTP Server', 'Port 8090', ['ws']);
  addNode(g, 150, 365, 100, 40, 'rgba(6,78,59,0.4)', '#34d399', 'Lobby', 'State Holder', 'Clubs, Citizens', ['svc']);
  addNode(g, 270, 365, 100, 40, 'rgba(6,78,59,0.4)', '#34d399', 'serveWs()', 'WebSocket', 'Gorilla WS', []);
  addNode(g, 390, 365, 100, 40, 'rgba(6,78,59,0.4)', '#34d399', 'API Routes', 'REST', '50+ routes', []);
  addNode(g, 510, 365, 100, 40, 'rgba(6,78,59,0.4)', '#34d399', 'Persistence', 'Gzip', 'Atomic .tmp', []);
  addNode(g, 630, 365, 100, 40, 'rgba(6,78,59,0.4)', '#34d399', 'Services', '30+', 'Battle, Economy', []);

  // Layer 4: Data
  g.appendChild(makeRect(15, 500, 1070, 170, 'rgba(120,53,15,0.1)', '#fbbf24', 10));
  g.appendChild(makeText(30, 515, 'DATA & STORAGE', 9, '#fbbf24', '700'));
  addNode(g, 30, 530, 100, 40, 'rgba(120,53,15,0.3)', '#fbbf24', 'localStorage', 'Client', 'Prefs, Beacon', []);
  addNode(g, 150, 530, 100, 40, 'rgba(120,53,15,0.3)', '#fbbf24', 'devdata/', 'Server Data', 'Gzip persistence', []);
  addNode(g, 270, 530, 100, 40, 'rgba(120,53,15,0.3)', '#fbbf24', 'WebSocket', 'Real-time', 'Bidirectional', []);
  addNode(g, 390, 530, 100, 40, 'rgba(120,53,15,0.3)', '#fbbf24', 'Beacon', 'Warm Boot', 'Session recovery', []);
  addNode(g, 510, 530, 100, 40, 'rgba(120,53,15,0.3)', '#fbbf24', 'Board Hash', 'Anti-cheat', 'SHA-256', []);
  addNode(g, 630, 530, 100, 40, 'rgba(120,53,15,0.3)', '#fbbf24', 'Public/', 'Static Assets', 'Images, Audio', []);

  // Cross-layer
  g.appendChild(makeLine(310, 115, 310, 228, '#a78bfa', '', true));
  g.appendChild(makeLine(545, 115, 545, 228, '#a78bfa', '', true));
  g.appendChild(makeLine(200, 270, 200, 363, '#34d399', '', true));
}

// ============================================================================
// VIEW: Server
// ============================================================================
function viewServer(g, data) {
  
  svg.setAttribute('viewBox', '0 0 1100 600');

  const defs = document.createElementNS('http://www.w3.org/2000/svg', 'defs');
  defs.innerHTML = '<pattern id="gridS" width="40" height="40" patternUnits="userSpaceOnUse"><path d="M 40 0 L 0 0 0 40" fill="none" stroke="#1e293b" stroke-width="0.5"/></pattern>';
  g.appendChild(defs);
  g.appendChild(makeRect(0, 0, 1100, 600, 'url(#gridS)', 'none'));

  g.appendChild(makeText(550, 25, 'Server Layer (server.go) — Build Tag: !js && !wasm', 11, 'white', '700'));

  addNode(g, 450, 50, 200, 50, 'rgba(6,78,59,0.4)', '#34d399', 'net/http Server', 'Port 8090', 'CORS, static files, API routing', ['lobby']);
  addNode(g, 450, 130, 200, 60, 'rgba(6,78,59,0.4)', '#34d399', 'Lobby Struct', 'Single Source of Truth', 'Clubs, Citizens, Religions, sync.RWMutex', ['svc', 'ws']);

  // Services
  addNode(g, 50, 220, 120, 40, 'rgba(6,78,59,0.4)', '#34d399', 'Battle Service', 'Combat', '3x3 grid, capture rules', []);
  addNode(g, 190, 220, 120, 40, 'rgba(6,78,59,0.4)', '#34d399', 'Economy Service', 'Trading', 'AMM bonding curve', []);
  addNode(g, 330, 220, 120, 40, 'rgba(6,78,59,0.4)', '#34d399', 'Justice System', 'Bounties', 'Wanted levels, missions', []);
  addNode(g, 470, 220, 120, 40, 'rgba(6,78,59,0.4)', '#34d399', 'Faith Service', 'Religion', '24 religions, rituals', []);
  addNode(g, 610, 220, 120, 40, 'rgba(6,78,59,0.4)', '#34d399', 'Career Service', 'Pathways', '12 careers, 5 tiers', []);
  addNode(g, 750, 220, 120, 40, 'rgba(6,78,59,0.4)', '#34d399', 'AI Citizens', 'BehavioralTick', '10 behaviors/citizen', []);

  // WebSocket + API
  addNode(g, 450, 300, 200, 50, 'rgba(136,19,55,0.4)', '#fb7185', 'serveWs()', 'Gorilla WebSocket', 'Upgrade, register, broadcast', []);

  addNode(g, 100, 390, 120, 40, 'rgba(6,78,59,0.4)', '#34d399', '/api/ai/citizens/*', 'AI CRUD', 'spawn, marry, breed', []);
  addNode(g, 240, 390, 120, 40, 'rgba(6,78,59,0.4)', '#34d399', '/api/faith/*', 'Religion', 'coherence, war-gambit', []);
  addNode(g, 380, 390, 120, 40, 'rgba(6,78,59,0.4)', '#34d399', '/api/rivalry/*', 'Rivalry', 'state, detect', []);
  addNode(g, 520, 390, 120, 40, 'rgba(6,78,59,0.4)', '#34d399', '/api/career/*', 'Career', 'progress, XP', []);
  addNode(g, 660, 390, 120, 40, 'rgba(6,78,59,0.4)', '#34d399', '/api/player/*', 'Profile', 'progression, stats', []);

  // Persistence
  addNode(g, 450, 480, 200, 50, 'rgba(120,53,15,0.3)', '#fbbf24', 'Gzip Persistence', 'Atomic Commits', '.tmp + rename, 5 backups', []);

  g.appendChild(makeLine(550, 100, 550, 128, '#34d399'));
  g.appendChild(makeLine(110, 200, 110, 218, '#34d399'));
  g.appendChild(makeLine(250, 200, 250, 218, '#34d399'));
  g.appendChild(makeLine(390, 200, 390, 218, '#34d399'));
  g.appendChild(makeLine(530, 200, 530, 218, '#34d399'));
  g.appendChild(makeLine(670, 200, 670, 218, '#34d399'));
  g.appendChild(makeLine(810, 200, 810, 218, '#34d399'));
  g.appendChild(makeLine(550, 190, 550, 298, '#fb7185'));
}

// ============================================================================
// VIEW: WASM Bridge
// ============================================================================
function viewWasm(g, data) {
  
  svg.setAttribute('viewBox', '0 0 1100 600');

  const defs = document.createElementNS('http://www.w3.org/2000/svg', 'defs');
  defs.innerHTML = '<pattern id="gridW" width="40" height="40" patternUnits="userSpaceOnUse"><path d="M 40 0 L 0 0 0 40" fill="none" stroke="#1e293b" stroke-width="0.5"/></pattern>';
  g.appendChild(defs);
  g.appendChild(makeRect(0, 0, 1100, 600, 'url(#gridW)', 'none'));

  g.appendChild(makeText(550, 25, 'WASM Layer (main.go) — Build Tag: js && wasm', 11, 'white', '700'));

  addNode(g, 450, 50, 200, 60, 'rgba(76,29,149,0.4)', '#a78bfa', 'Engine Struct', 'State Machine', 'Players[2], Board[9], Clubs, Citizens', ['bridge']);
  addNode(g, 450, 140, 200, 50, 'rgba(76,29,149,0.4)', '#a78bfa', 'registerWasmHooks()', '50+ Functions', 'js.Global().Set() pattern', ['exp', 'ing']);

  // Exporters
  addNode(g, 80, 230, 120, 40, 'rgba(76,29,149,0.4)', '#a78bfa', 'GetGameState()', 'Read', 'Filter: all/profile/combat/economy', []);
  addNode(g, 220, 230, 120, 40, 'rgba(76,29,149,0.4)', '#a78bfa', 'GetMenuState()', 'Read', 'Menu/controller JSON', []);
  addNode(g, 360, 230, 120, 40, 'rgba(76,29,149,0.4)', '#a78bfa', 'SyncMove()', 'Battle', 'Board hash validation', []);

  // Ingesters
  addNode(g, 560, 230, 120, 40, 'rgba(76,29,149,0.4)', '#a78bfa', 'SyncFullProfile()', 'Write', 'Reputation, wins, mojo', []);
  addNode(g, 700, 230, 120, 40, 'rgba(76,29,149,0.4)', '#a78bfa', 'SetMenuLayout()', 'Write', 'Grid, shape, size', []);
  addNode(g, 840, 230, 120, 40, 'rgba(76,29,149,0.4)', '#a78bfa', 'ToggleStarItem()', 'Write', 'Add/remove favorite', []);

  // Menu bridge
  addNode(g, 80, 320, 120, 40, 'rgba(76,29,149,0.4)', '#a78bfa', 'HandleControllerInput()', 'Nav', 'Returns focus index', []);
  addNode(g, 220, 320, 120, 40, 'rgba(76,29,149,0.4)', '#a78bfa', 'SetControllerType()', 'Gamepad', 'xbox/ps/nintendo', []);
  addNode(g, 360, 320, 120, 40, 'rgba(76,29,149,0.4)', '#a78bfa', 'SetButtonOverride()', 'Per-button', 'Shape/size', []);

  // Replay + Hash
  addNode(g, 550, 410, 200, 50, 'rgba(76,29,149,0.4)', '#a78bfa', 'ClientReplayEngine', 'Catch-up', '100 frame buffer, SHA-256', []);
  addNode(g, 550, 490, 200, 50, 'rgba(76,29,149,0.4)', '#a78bfa', 'ComputeBoardHash()', 'Anti-cheat', 'SHA-256 of board state', []);

  g.appendChild(makeLine(550, 110, 550, 138, '#a78bfa'));
  g.appendChild(makeLine(550, 190, 140, 228, '#a78bfa'));
  g.appendChild(makeLine(550, 190, 280, 228, '#a78bfa'));
  g.appendChild(makeLine(550, 190, 420, 228, '#a78bfa'));
  g.appendChild(makeLine(550, 190, 620, 228, '#a78bfa'));
  g.appendChild(makeLine(550, 190, 760, 228, '#a78bfa'));
  g.appendChild(makeLine(550, 190, 900, 228, '#a78bfa'));
  g.appendChild(makeLine(550, 270, 550, 408, '#a78bfa'));
  g.appendChild(makeLine(550, 460, 550, 488, '#a78bfa'));
}

// ============================================================================
// VIEW: JavaScript (data-driven by category)
// ============================================================================
function viewJS(g, data) {
  
  svg.setAttribute('viewBox', '0 0 1100 700');

  const defs = document.createElementNS('http://www.w3.org/2000/svg', 'defs');
  defs.innerHTML = '<pattern id="gridJ" width="40" height="40" patternUnits="userSpaceOnUse"><path d="M 40 0 L 0 0 0 40" fill="none" stroke="#1e293b" stroke-width="0.5"/></pattern>';
  g.appendChild(defs);
  g.appendChild(makeRect(0, 0, 1100, 700, 'url(#gridJ)', 'none'));

  const jsModules = MANIFEST_DATA.jsModules;
  g.appendChild(makeText(550, 25, `JavaScript Layer — ${jsModules.length} Modules (${MANIFEST_DATA.summary.esModules} ES)`, 11, 'white', '700'));

  // app.js hub
  addNode(g, 450, 50, 200, 50, 'rgba(8,51,68,0.4)', '#22d3ee', 'app.js', 'Orchestrator', 'Imports all, boots WASM, window.* bridge', ['core', 'game', 'econ', 'social', 'menu', 'ai']);

  // Data-driven category groups
  const groups = groupModulesByCategory(jsModules);
  const categoryColors = {
    core: '#22d3ee', game: '#34d399', economy: '#fbbf24',
    social: '#fb7185', menu: '#a78bfa', ai: '#22d3ee', other: '#64748b'
  };

  let yOffset = 150;
  let catIndex = 0;
  const catIds = ['core', 'game', 'economy', 'social', 'menu', 'ai', 'other'];

  for (const catId of catIds) {
    const mods = groups[catId];
    if (!mods || mods.length === 0) continue;

    const color = categoryColors[catId] || '#64748b';
    g.appendChild(makeText(30, yOffset - 10, catId.toUpperCase(), 8, '#94a3b8', '600'));

    for (let i = 0; i < mods.length; i++) {
      const mod = mods[i];
      const x = 30 + (i * 110);
      addNode(g, x, yOffset, 90, 35, 'rgba(8,51,68,0.4)', color, mod.fileName, `${mod.exportCount} exports`, mod.isESModule ? 'ES Module' : 'IIFE', []);
    }

    yOffset += 60;
    catIndex++;
  }

  // Connections from app.js
  catIds.forEach((catId, idx) => {
    if (groups[catId] && groups[catId].length > 0) {
      const y = 150 + idx * 60;
      g.appendChild(makeLine(450, 100, 75, y - 2, '#22d3ee'));
    }
  });
}

// ============================================================================
// VIEW: CSS/SCSS
// ============================================================================
function viewCSS(g, data) {
  
  svg.setAttribute('viewBox', '0 0 1100 500');

  const defs = document.createElementNS('http://www.w3.org/2000/svg', 'defs');
  defs.innerHTML = '<pattern id="gridC" width="40" height="40" patternUnits="userSpaceOnUse"><path d="M 40 0 L 0 0 0 40" fill="none" stroke="#1e293b" stroke-width="0.5"/></pattern>';
  g.appendChild(defs);
  g.appendChild(makeRect(0, 0, 1100, 500, 'url(#gridC)', 'none'));

  g.appendChild(makeText(550, 25, 'CSS/SCSS Layer — Compiled to Public/css/main.css', 11, 'white', '700'));

  addNode(g, 450, 50, 200, 50, 'rgba(120,53,15,0.3)', '#fbbf24', 'main.scss', 'Entry Point', 'Imports 50+ partials', ['base', 'themes', 'layouts', 'components', 'utilities']);

  addNode(g, 80, 140, 120, 40, 'rgba(120,53,15,0.3)', '#fbbf24', 'base/', 'Foundation', 'variables, reset', []);
  addNode(g, 220, 140, 120, 40, 'rgba(120,53,15,0.3)', '#fbbf24', 'themes/', 'Theming', 'neon-glass', []);
  addNode(g, 360, 140, 120, 40, 'rgba(120,53,15,0.3)', '#fbbf24', 'layouts/', 'Layout', 'main-layout', []);
  addNode(g, 500, 140, 120, 40, 'rgba(120,53,15,0.3)', '#fbbf24', 'components/', 'Components', 'buttons, cards', []);
  addNode(g, 640, 140, 120, 40, 'rgba(120,53,15,0.3)', '#fbbf24', 'utilities/', 'Utilities', 'spacing', []);

  g.appendChild(makeText(550, 210, 'Feature Partials', 9, '#94a3b8', '600'));

  const scssPartials = MANIFEST_DATA.scssPartials || [];
  const topPartials = scssPartials.slice(0, 5);
  topPartials.forEach((partial, i) => {
    addNode(g, 80 + i * 150, 230, 130, 40, 'rgba(120,53,15,0.3)', '#fbbf24', partial.fileName, partial.ruleCount + ' rules', 'partial', []);
  });

  g.appendChild(makeLine(550, 100, 140, 138, '#fbbf24'));
  g.appendChild(makeLine(550, 100, 280, 138, '#fbbf24'));
  g.appendChild(makeLine(550, 100, 420, 138, '#fbbf24'));
  g.appendChild(makeLine(550, 100, 560, 138, '#fbbf24'));
  g.appendChild(makeLine(550, 100, 700, 138, '#fbbf24'));
}

// ============================================================================
// VIEW: Data Flow
// ============================================================================
function viewData(g, data) {
  
  svg.setAttribute('viewBox', '0 0 1100 600');

  const defs = document.createElementNS('http://www.w3.org/2000/svg', 'defs');
  defs.innerHTML = '<pattern id="gridD" width="40" height="40" patternUnits="userSpaceOnUse"><path d="M 40 0 L 0 0 0 40" fill="none" stroke="#1e293b" stroke-width="0.5"/></pattern>';
  g.appendChild(defs);
  g.appendChild(makeRect(0, 0, 1100, 600, 'url(#gridD)', 'none'));

  g.appendChild(makeText(550, 25, 'Data Flow — User Action to State Update', 11, 'white', '700'));

  addNode(g, 80, 60, 120, 40, 'rgba(8,51,68,0.4)', '#22d3ee', 'User Click', 'DOM Event', 'Button, card, tile', ['js']);
  addNode(g, 240, 60, 120, 40, 'rgba(8,51,68,0.4)', '#22d3ee', 'JS Handler', 'app.js bridge', 'window.handleAction()', ['wasm']);
  addNode(g, 400, 60, 120, 40, 'rgba(76,29,149,0.4)', '#a78bfa', 'WASM Bridge', 'js.FuncOf()', 'window.SetX()', ['server']);
  addNode(g, 560, 60, 120, 40, 'rgba(6,78,59,0.4)', '#34d399', 'Server Validate', 'Lobby mutex', 'Authoritative check', ['broadcast']);
  addNode(g, 720, 60, 120, 40, 'rgba(6,78,59,0.4)', '#34d399', 'Broadcast', 'WebSocket', 'lobby_update', ['wasm2']);

  addNode(g, 400, 150, 120, 40, 'rgba(76,29,149,0.4)', '#a78bfa', 'WASM State', 'Engine struct', 'Updated by SyncX()', ['syncui']);
  addNode(g, 240, 150, 120, 40, 'rgba(8,51,68,0.4)', '#22d3ee', 'syncUI()', 'Render Loop', 'Reads GetGameState()', ['dom']);
  addNode(g, 80, 150, 120, 40, 'rgba(8,51,68,0.4)', '#22d3ee', 'DOM Update', 'React-like', 'innerHTML, classList', []);

  g.appendChild(makeLine(200, 80, 238, 80, '#64748b'));
  g.appendChild(makeLine(360, 80, 398, 80, '#64748b'));
  g.appendChild(makeLine(520, 80, 558, 80, '#64748b'));
  g.appendChild(makeLine(680, 80, 718, 80, '#64748b'));
  g.appendChild(makeLine(780, 100, 780, 130, '#64748b'));
  g.appendChild(makeLine(780, 130, 522, 168, '#64748b'));
  g.appendChild(makeLine(400, 190, 362, 190, '#64748b'));
  g.appendChild(makeLine(240, 190, 202, 190, '#64748b'));

  // Server push
  g.appendChild(makeText(550, 270, 'Server Push Flow', 9, '#94a3b8', '600'));
  addNode(g, 600, 290, 120, 40, 'rgba(6,78,59,0.4)', '#34d399', 'Server Event', 'State Change', 'AI tick, market', ['ws']);
  addNode(g, 440, 290, 120, 40, 'rgba(136,19,55,0.4)', '#fb7185', 'WebSocket', 'Broadcast', 'lobby_update', ['net']);
  addNode(g, 280, 290, 120, 40, 'rgba(8,51,68,0.4)', '#22d3ee', 'network.js', 'Handler', 'Parse + dispatch', ['sync']);
  addNode(g, 120, 290, 120, 40, 'rgba(76,29,149,0.4)', '#a78bfa', 'WASM Sync', 'SyncX()', 'Update Engine', []);

  g.appendChild(makeLine(720, 310, 760, 310, '#fb7185'));
  g.appendChild(makeLine(560, 310, 522, 310, '#64748b'));
  g.appendChild(makeLine(400, 310, 362, 310, '#64748b'));
  g.appendChild(makeLine(240, 310, 218, 310, '#64748b'));

  // Warm boot
  g.appendChild(makeText(550, 380, 'Warm Boot (Beacon Recovery)', 9, '#94a3b8', '600'));
  addNode(g, 280, 400, 120, 40, 'rgba(120,53,15,0.3)', '#fbbf24', 'localStorage', 'Beacon', 'Last known state', ['app']);
  addNode(g, 440, 400, 120, 40, 'rgba(8,51,68,0.4)', '#22d3ee', 'app.js onload', 'Beacon parse', 'JSON.parse()', ['restore']);
  addNode(g, 600, 400, 120, 40, 'rgba(76,29,149,0.4)', '#a78bfa', 'WASM Restore', 'SyncX()', 'Rehydrate Engine', []);

  g.appendChild(makeLine(400, 420, 438, 420, '#fbbf24'));
  g.appendChild(makeLine(560, 420, 598, 420, '#64748b'));
}

// ============================================================================
// VIEW: Analytics (with sub-tabs)
// ============================================================================
function viewAnalytics(g, data) {
  
  svg.setAttribute('viewBox', '0 0 1100 600');

  const defs = document.createElementNS('http://www.w3.org/2000/svg', 'defs');
  defs.innerHTML = '<pattern id="gridA" width="40" height="40" patternUnits="userSpaceOnUse"><path d="M 40 0 L 0 0 0 40" fill="none" stroke="#1e293b" stroke-width="0.5"/></pattern>';
  g.appendChild(defs);
  g.appendChild(makeRect(0, 0, 1100, 600, 'url(#gridA)', 'none'));

  g.appendChild(makeText(550, 25, 'Architecture Analytics', 11, 'white', '700'));

  // Sub-tabs
  subTabContainer.innerHTML = '<div class="sub-tabs"><button class="sub-tab active" data-sub="overview">Overview</button><button class="sub-tab" data-sub="language">Language</button><button class="sub-tab" data-sub="connections">Connections</button><button class="sub-tab" data-sub="health">Health</button><button class="sub-tab" data-sub="wired">🔌 Wired</button></div>';

  // Render overview by default
  renderOverviewChart(g);

  // Sub-tab handlers
  document.querySelectorAll('.sub-tab').forEach(tab => {
    tab.addEventListener('click', () => {
      document.querySelectorAll('.sub-tab').forEach(t => t.classList.remove('active'));
      tab.classList.add('active');
      currentAnalyticsTab = tab.dataset.sub;
      renderAnalyticsSub(currentAnalyticsTab);
    });
  });
}

// ============================================================================
// ANALYTICS SUB-TABS
// ============================================================================

function renderAnalyticsSub(tab) {
  svg.innerHTML = '';
  const g = document.createElementNS('http://www.w3.org/2000/svg', 'g');
  svg.appendChild(g);

  const defs = document.createElementNS('http://www.w3.org/2000/svg', 'defs');
  defs.innerHTML = '<pattern id="gridA" width="40" height="40" patternUnits="userSpaceOnUse"><path d="M 40 0 L 0 0 0 40" fill="none" stroke="#1e293b" stroke-width="0.5"/></pattern>';
  g.appendChild(defs);
  g.appendChild(makeRect(0, 0, 1100, 600, 'url(#gridA)', 'none'));

  if (tab === 'overview') renderOverviewChart(g);
  else if (tab === 'language') renderLanguageChart(g);
  else if (tab === 'connections') renderConnectionsChart(g);
  else if (tab === 'health') renderHealthChart(g);
  else if (tab === 'wired') renderWiredChart(g);
}

function renderOverviewChart(g) {
  svg.innerHTML = '';
  const newG = document.createElementNS('http://www.w3.org/2000/svg', 'g');
  svg.appendChild(newG);

  const defs = document.createElementNS('http://www.w3.org/2000/svg', 'defs');
  defs.innerHTML = '<pattern id="gridA" width="40" height="40" patternUnits="userSpaceOnUse"><path d="M 40 0 L 0 0 0 40" fill="none" stroke="#1e293b" stroke-width="0.5"/></pattern>';
  newG.appendChild(defs);
  newG.appendChild(makeRect(0, 0, 1100, 600, 'url(#gridA)', 'none'));

  newG.appendChild(makeText(550, 25, 'Architecture Analytics', 11, 'white', '700'));
  newG.appendChild(makeText(550, 70, 'Quick Overview', 10, 'white', '600'));

  const ad = ANALYTICS_DATA;
  const overviewData = [
    { label: 'Bridge Functions', value: ad.summary.bridgeFunctions, color: '#a78bfa' },
    { label: 'ES Modules', value: ad.summary.esModules, color: '#22d3ee' },
    { label: 'IIFE Legacy', value: ad.summary.iifeModules, color: '#64748b' },
    { label: 'HTML Scripts', value: ad.summary.htmlScripts, color: '#fbbf24' },
    { label: 'SCSS Partials', value: ad.summary.scssPartials, color: '#34d399' },
    { label: 'Total Files', value: ad.totalFiles, color: '#fb7185' }
  ];
  const overviewPie = makePie(overviewData, 140);
  const og = document.createElementNS('http://www.w3.org/2000/svg', 'g');
  og.setAttribute('transform', 'translate(380,90)');
  og.innerHTML = overviewPie;
  newG.appendChild(og);
  newG.appendChild(makeText(450, 250, 'Files by Type', 8, '#94a3b8'));

  newG.appendChild(makeText(700, 90, 'Counts', 10, 'white', '600'));
  newG.appendChild(makeText(700, 115, 'Go Files: ' + ad.goFiles, 8, '#34d399'));
  newG.appendChild(makeText(700, 135, 'JS Files: ' + ad.jsFiles, 8, '#22d3ee'));
  newG.appendChild(makeText(700, 155, 'CSS Files: ' + ad.cssFiles, 8, '#fbbf24'));
  newG.appendChild(makeText(700, 175, 'Total Exports: ' + ad.totalExports, 8, '#94a3b8'));
  newG.appendChild(makeText(700, 195, 'Total Imports: ' + ad.totalImports, 8, '#94a3b8'));
  newG.appendChild(makeText(700, 215, 'Window Bindings: ' + ad.totalWindowBindings, 8, '#94a3b8'));
}

function renderWiredChart(g) {
  const stats = WIRED_DATA.stats;
  g.appendChild(makeText(550, 30, '🔌 Wired vs Orphaned Files', 10, 'white', '600'));
  g.appendChild(makeText(550, 60, 'Total Files: ' + stats.total, 9, '#22d3ee'));
  g.appendChild(makeText(550, 80, 'Wired: ' + stats.wired + ' (' + stats.wiredPercent + '%)', 8, '#34d399'));
  g.appendChild(makeText(550, 100, 'Orphaned: ' + stats.orphaned + ' (' + (100 - stats.wiredPercent) + '%)', 8, '#fb7185'));

  const data = [
    { label: 'Wired', value: stats.wired, color: '#34d399' },
    { label: 'Orphaned', value: stats.orphaned, color: '#fb7185' }
  ];
  const pie = makePie(data, 120);
  const pg = document.createElementNS('http://www.w3.org/2000/svg', 'g');
  pg.setAttribute('transform', 'translate(420,130)');
  pg.innerHTML = pie;
  g.appendChild(pg);

  g.appendChild(makeText(750, 130, 'By Category', 9, 'white', '600'));
  let catY = 150;
  Object.entries(WIRED_DATA.byCategory)
    .sort((a, b) => (b[1].wired + b[1].orphaned) - (a[1].wired + a[1].orphaned))
    .slice(0, 8)
    .forEach(([cat, counts]) => {
      const total = counts.wired + counts.orphaned;
      const pct = Math.round((counts.wired / total) * 100);
      g.appendChild(makeText(750, catY, cat + ': ' + counts.wired + '/' + total + ' (' + pct + '%)', 7, '#94a3b8'));
      catY += 18;
    });

  g.appendChild(makeText(100, 280, 'Orphaned Files (sample)', 9, '#fb7185', '600'));
  const orphanedSample = (WIRED_DATA.orphaned || []).slice(0, 15);
  let orphanY = 300;
  orphanedSample.forEach(f => {
    const name = typeof f === 'string' ? f : (f && f.path) || '';
    g.appendChild(makeText(100, orphanY, String(name).substring(0, 60), 7, '#94a3b8'));
    orphanY += 16;
  });
  if ((WIRED_DATA.orphaned || []).length > 15) {
    g.appendChild(makeText(100, orphanY, '... and ' + (WIRED_DATA.orphaned.length - 15) + ' more', 7, '#64748b'));
  }
}

function formatBytes(n) {
  if (n >= 1048576) return (n / 1048576).toFixed(1) + ' MB';
  if (n >= 1024) return (n / 1024).toFixed(1) + ' KB';
  return n + ' B';
}

function languageInventory() {
  if (ANALYTICS_DATA.languages && ANALYTICS_DATA.languages.length) {
    return ANALYTICS_DATA.languages;
  }
  const files = [...(WIRED_DATA.wired || []), ...(WIRED_DATA.orphaned || [])];
  const rules = [
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
  const buckets = {};
  for (const f of files) {
    const p = f.path || '';
    let label = 'Other';
    for (const [rx, name] of rules) { if (rx.test(p)) { label = name; break; } }
    if (!buckets[label]) buckets[label] = { label, files: 0, bytes: 0 };
    buckets[label].files++;
    buckets[label].bytes += f.size || 0;
  }
  return Object.values(buckets).sort((a, b) => b.files - a.files);
}

function renderLanguageChart(g) {
  const COLORS = {
    Go: '#34d399', JavaScript: '#22d3ee', TypeScript: '#60a5fa',
    CSS: '#fbbf24', HTML: '#fb7185', Markdown: '#a78bfa',
    JSON: '#94a3b8', Python: '#38bdf8', Shell: '#64748b',
    Images: '#f472b6', Media: '#c084fc', Other: '#475569'
  };
  const langs = languageInventory();
  const totalFiles = langs.reduce((s, l) => s + l.files, 0) || 1;
  const totalBytes = langs.reduce((s, l) => s + (l.bytes || 0), 0);

  g.appendChild(makeText(550, 30, 'Language Distribution', 10, 'white', '600'));
  g.appendChild(makeText(550, 50, totalFiles + ' files  ·  ' + formatBytes(totalBytes) + '  (repo inventory)', 8, '#94a3b8'));

  const data = langs.map(l => ({
    label: l.label,
    value: l.files,
    color: COLORS[l.label] || '#475569'
  }));
  const pie = makePie(data, 140);
  const pg = document.createElementNS('http://www.w3.org/2000/svg', 'g');
  pg.setAttribute('transform', 'translate(80,80)');
  pg.innerHTML = pie;
  g.appendChild(pg);

  g.appendChild(makeText(150, 240, 'By file count', 8, '#64748b'));

  g.appendChild(makeText(420, 80, 'Language', 8, '#64748b', '600', 'start'));
  g.appendChild(makeText(620, 80, 'Files', 8, '#64748b', '600', 'end'));
  g.appendChild(makeText(700, 80, '%', 8, '#64748b', '600', 'end'));
  g.appendChild(makeText(820, 80, 'Size', 8, '#64748b', '600', 'end'));

  let y = 102;
  langs.forEach(l => {
    const color = COLORS[l.label] || '#475569';
    const pct = Math.round((l.files / totalFiles) * 100);
    g.appendChild(makeRect(400, y - 8, 8, 8, color, 'none', 2));
    g.appendChild(makeText(420, y, l.label, 8, color, '600', 'start'));
    g.appendChild(makeText(620, y, String(l.files), 8, '#e2e8f0', '600', 'end'));
    g.appendChild(makeText(700, y, pct + '%', 8, '#94a3b8', '400', 'end'));
    g.appendChild(makeText(820, y, formatBytes(l.bytes || 0), 8, '#94a3b8', '400', 'end'));
    const barW = Math.max(2, Math.round((l.files / totalFiles) * 280));
    g.appendChild(makeRect(400, y + 6, 280, 4, 'rgba(30,41,59,0.6)', 'none', 2));
    g.appendChild(makeRect(400, y + 6, barW, 4, color, 'none', 2));
    y += 32;
  });

  g.appendChild(makeText(550, y + 16, 'Source: wired inventory (every scanned repo file by extension)', 7, '#64748b'));
}

function renderConnectionsChart(g) {
  const ad = ANALYTICS_DATA;
  g.appendChild(makeText(550, 30, 'Connection Density', 10, 'white', '600'));
  g.appendChild(makeText(550, 70, 'Total Connections: ' + (ad.totalImports + ad.totalWindowBindings), 9, '#22d3ee'));

  const data = [
    { label: 'Imports', value: ad.totalImports, color: '#22d3ee' },
    { label: 'Window Bindings', value: ad.totalWindowBindings, color: '#a78bfa' }
  ];
  const pie = makePie(data, 120);
  const pg = document.createElementNS('http://www.w3.org/2000/svg', 'g');
  pg.setAttribute('transform', 'translate(420,100)');
  pg.innerHTML = pie;
  g.appendChild(pg);

  g.appendChild(makeText(480, 240, 'Imports: ' + ad.totalImports, 8, '#22d3ee'));
  g.appendChild(makeText(480, 260, 'Bindings: ' + ad.totalWindowBindings, 8, '#a78bfa'));

  g.appendChild(makeText(750, 100, 'Avg per Module', 9, 'white', '600'));
  g.appendChild(makeText(750, 120, 'Imports: ' + (ad.totalImports / Math.max(ad.summary.esModules, 1)).toFixed(1), 8, '#94a3b8'));
  g.appendChild(makeText(750, 140, 'Exports: ' + (ad.totalExports / Math.max(ad.summary.esModules, 1)).toFixed(1), 8, '#94a3b8'));
  g.appendChild(makeText(750, 160, 'Bindings: ' + (ad.totalWindowBindings / Math.max(ad.summary.esModules, 1)).toFixed(1), 8, '#94a3b8'));
}

/**
 * HEALTH CHART — The core architecture health visualization.
 *
 * Health is calculated from 4 factors:
 *   1. Wired ratio: wired files / total files (are files connected?)
 *   2. Flow integrity: complete bridge flows / total bridge flows (registration + handler paired?)
 *   3. Module connectedness: ES modules with at least 1 import or export
 *   4. Build tag violations: malformed build tags
 *
 * Each factor contributes 25 points to the 100-point score.
 */
function renderHealthChart(g) {
  const ad = ANALYTICS_DATA;
  const wired = WIRED_DATA;
  const summary = MANIFEST_DATA.summary;
  const flowIntegrity = MANIFEST_DATA.flowIntegrity;

  // ===== CALCULATE HEALTH SCORE =====
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
  const connectedModules = MANIFEST_DATA.jsModules.filter(m => m.connectionCount > 0).length;
  const connectednessRatio = connectedModules / Math.max(MANIFEST_DATA.jsModules.length, 1);
  const connectednessScore = Math.round(connectednessRatio * 25);

  // Factor 4: Build tag violations (0-25 points, inverted)
  // No violations = full points, each violation costs 5 points
  const violationPenalty = Math.min(25, (summary.violations || 0) * 5);
  const violationScore = 25 - violationPenalty;

  // Total health
  const healthScore = wiredScore + integrityScore + connectednessScore + violationScore;

  // Render
  const color = healthScore >= 80 ? '#34d399' : healthScore >= 50 ? '#fbbf24' : '#fb7185';
  g.appendChild(makeText(550, 30, 'App Health Score', 10, 'white', '600'));
  g.appendChild(makeText(550, 80, healthScore + '%', 24, color));

  // Health bar
  g.appendChild(makeRect(350, 100, 400, 20, 'rgba(30,41,59,0.5)', '#1e293b', 10));
  g.appendChild(makeRect(350, 100, healthScore * 4, 20, color, 'none', 10));

  g.appendChild(makeText(550, 150, (healthScore >= 80 ? 'Healthy' : healthScore >= 50 ? 'Needs Attention' : 'Critical'), 10, color));

  // Score breakdown
  g.appendChild(makeText(550, 190, 'Score Breakdown', 9, 'white', '600'));
  g.appendChild(makeText(550, 215, `Wired Ratio: ${wiredScore}/25 (${Math.round(wiredRatio * 100)}% wired)`, 8, '#94a3b8'));
  g.appendChild(makeText(550, 235, `Flow Integrity: ${integrityScore}/25 (${flowIntegrity.integrityPercent}% complete)`, 8, '#94a3b8'));
  g.appendChild(makeText(550, 255, `Module Connectedness: ${connectednessScore}/25 (${connectedModules}/${MANIFEST_DATA.jsModules.length} connected)`, 8, '#94a3b8'));
  g.appendChild(makeText(550, 275, `Build Violations: ${violationScore}/25 (${summary.violations || 0} violations)`, 8, '#94a3b8'));

  // Orphaned files detail (the "flow health" indicator)
  g.appendChild(makeText(550, 310, 'Flow Health Indicators', 9, 'white', '600'));
  g.appendChild(makeText(550, 335, `Loose/Orphaned Files: ${wired.stats.orphaned} (lower is better)`, 8, wired.stats.orphaned > 50 ? '#fb7185' : '#34d399'));
  g.appendChild(makeText(550, 355, `Incomplete Bridge Flows: ${flowIntegrity.registrationOnly + flowIntegrity.handlerOnly}`, 8, '#94a3b8'));
  g.appendChild(makeText(550, 375, `Registration-only: ${flowIntegrity.registrationOnly} (exposed but no handler)`, 8, '#94a3b8'));
  g.appendChild(makeText(550, 395, `Handler-only: ${flowIntegrity.handlerOnly} (handled but not registered)`, 8, '#94a3b8'));

  // Recommendations
  g.appendChild(makeText(550, 430, 'Recommendations', 9, 'white', '600'));
  let recY = 455;
  if (wired.stats.orphaned > 50) {
    g.appendChild(makeText(550, recY, `• ${wired.stats.orphaned} orphaned files need attention`, 8, '#fb7185'));
    recY += 20;
  }
  if (flowIntegrity.registrationOnly > 0) {
    g.appendChild(makeText(550, recY, `• ${flowIntegrity.registrationOnly} bridge functions registered but not implemented`, 8, '#fbbf24'));
    recY += 20;
  }
  if (flowIntegrity.handlerOnly > 0) {
    g.appendChild(makeText(550, recY, `• ${flowIntegrity.handlerOnly} handlers not registered via js.Global().Set()`, 8, '#fbbf24'));
    recY += 20;
  }
  if (summary.violations > 0) {
    g.appendChild(makeText(550, recY, `• Fix ${summary.violations} build tag violations`, 8, '#fb7185'));
    recY += 20;
  }
  if (recY === 455) {
    g.appendChild(makeText(550, recY, '✓ All checks passing!', 8, '#34d399'));
  }
}

// ============================================================================
// VIEW REGISTRATION (maps tab names to view functions)
// ============================================================================

views.full = viewFull;
views.server = viewServer;
views.wasm = viewWasm;
views.js = viewJS;
views.css = viewCSS;
views.data = viewData;
views.analytics = viewAnalytics;

// ============================================================================
// EXPORTS
// ============================================================================

module.exports = {
  viewFull,
  viewServer,
  viewWasm,
  viewJS,
  viewCSS,
  viewData,
  viewAnalytics,
  renderAnalyticsSub,
  renderOverviewChart,
  renderWiredChart,
  renderLanguageChart,
  renderConnectionsChart,
  renderHealthChart,
  getCategory,
  groupModulesByCategory,
  groupBridgesByPrefix,
  CATEGORY_PREFIXES
};
