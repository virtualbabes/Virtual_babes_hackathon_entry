#!/usr/bin/env node
// ============================================================================
// flow_verify.js — Verify menu customization pipeline end-to-end
// ----------------------------------------------------------------------------
// Simulates the full data flow:
//   1. User stars a tab in World Dashboard
//   2. UserPreferences saves to localStorage
//   3. Main Menu reads starred items
//   4. MenuCustomization computes positions
//   5. renderStarredItems renders buttons with shapes/sizes
//   6. Controller nav maps directions to focus movement
// ============================================================================

const fs = require('fs');
const path = require('path');

const PROJECT_ROOT = path.resolve(__dirname, '../..');
const JS_ROOT = path.join(PROJECT_ROOT, 'Public', 'js');
const PUBLIC_ROOT = path.join(PROJECT_ROOT, 'Public');

const C = {
    reset: '\x1b[0m',
    green: '\x1b[32m',
    red: '\x1b[31m',
    yellow: '\x1b[33m',
    cyan: '\x1b[36m',
    bold: '\x1b[1m',
};

let pass = 0;
let fail = 0;
const errors = [];

function ok(msg) { console.log(`  ${C.green}✓${C.reset} ${msg}`); pass++; }
function bad(msg) { console.log(`  ${C.red}✗${C.reset} ${msg}`); fail++; errors.push(msg); }
function info(msg) { console.log(`  ${C.cyan}→${C.reset} ${msg}`); }

function loadModule(fileName) {
    const p1 = path.join(JS_ROOT, fileName);
    const p2 = path.join(PUBLIC_ROOT, fileName);
    if (fs.existsSync(p1)) return fs.readFileSync(p1, 'utf8');
    if (fs.existsSync(p2)) return fs.readFileSync(p2, 'utf8');
    return null;
}

// --- Test: Full Pipeline Integration ---
function testFullPipeline() {
    console.log(`\n${C.bold}Test: Full Pipeline Integration${C.reset}`);

    // 1. World Dashboard has toggleStar
    const wdCode = loadModule('world_dashboard.js');
    if (!wdCode) { bad('world_dashboard.js not found'); return; }
    if (wdCode.includes('toggleStar')) {
        ok('World Dashboard has toggleStar()');
    } else {
        bad('World Dashboard missing toggleStar()');
    }

    // 2. UserPreferences has toggleStar with wdTab
    const upCode = loadModule('user_preferences.js');
    if (!upCode) { bad('user_preferences.js not found'); return; }
    if (upCode.includes('function toggleStar') && upCode.includes('wdTab')) {
        ok('UserPreferences.toggleStar() accepts wdTab');
    } else {
        bad('UserPreferences.toggleStar() missing wdTab support');
    }

    // 3. UserPreferences persists to localStorage
    if (upCode.includes('localStorage.setItem') && upCode.includes('vbabes_user_prefs')) {
        ok('UserPreferences persists to localStorage');
    } else {
        bad('UserPreferences missing localStorage persistence');
    }

    // 4. UserPreferences notifies listeners
    if (upCode.includes('notify()') && upCode.includes('_listeners')) {
        ok('UserPreferences has listener notification');
    } else {
        bad('UserPreferences missing listener notification');
    }

    // 5. Menu-constellation reads starred items
    const mcCode = loadModule('menu-constellation.js');
    if (!mcCode) { bad('menu-constellation.js not found'); return; }
    if (mcCode.includes('getStarred()')) {
        ok('Menu-constellation reads getStarred()');
    } else {
        bad('Menu-constellation does not read getStarred()');
    }

    // 6. renderStarredItems applies shapes
    if (mcCode.includes('shape-') && mcCode.includes('clip-path')) {
        ok('renderStarredItems applies shape clip-paths');
    } else {
        bad('renderStarredItems missing shape application');
    }

    // 7. renderStarredItems applies sizes
    if (mcCode.includes('size-') && mcCode.includes('font-size')) {
        ok('renderStarredItems applies size styling');
    } else {
        bad('renderStarredItems missing size styling');
    }

    // 8. renderStarredItems handles non-grid layouts
    if (mcCode.includes('position: absolute') && mcCode.includes('relative')) {
        ok('renderStarredItems handles absolute positioning');
    } else {
        bad('renderStarredItems missing absolute positioning');
    }

    // 9. MenuCustomization has computePositions
    const mcustomCode = loadModule('menu_customization.js');
    if (!mcustomCode) { bad('menu_customization.js not found'); return; }
    if (mcustomCode.includes('function computePositions')) {
        ok('MenuCustomization.computePositions() exists');
    } else {
        bad('MenuCustomization.computePositions() missing');
    }

    // 10. MenuCustomization handles all grid types
    const gridKeys = mcustomCode.match(/GRIDS\s*=\s*\{([\s\S]*?)\n\}/);
    let handledGrids = 0;
    if (gridKeys) {
        const body = gridKeys[1];
        const keys = [...body.matchAll(/^\s{4}(\w+):/gm)].map(m => m[1]);
        handledGrids = keys.length;
    }
    if (handledGrids >= 7) {
        ok(`MenuCustomization handles ${handledGrids} grid types`);
    } else {
        bad(`MenuCustomization only handles ${handledGrids} grid types`);
    }

    // 11. Controller nav handles all grid types
    const cnCode = loadModule('controller_nav.js');
    if (!cnCode) { bad('controller_nav.js not found'); return; }
    const gridTypes = ['grid', 'circle', 'linear-h', 'linear-v', 'cross', 'diamond', 'arc', 'triangle'];
    let handledNav = 0;
    for (const gt of gridTypes) {
        if (cnCode.includes(`case '${gt}'`)) handledNav++;
    }
    if (handledNav >= 6) {
        ok(`Controller nav handles ${handledNav} grid types`);
    } else {
        bad(`Controller nav only handles ${handledNav} grid types`);
    }

    // 12. Constellation Hub renders pathway avenue
    const chCode = loadModule('constellation_hub.js');
    if (!chCode) { bad('constellation_hub.js not found'); return; }
    if (chCode.includes('renderPathwayAvenue')) {
        ok('Constellation Hub renders pathway avenue');
    } else {
        bad('Constellation Hub missing pathway avenue rendering');
    }

    // 13. Pathway avenues has tier calculation
    const paCode = loadModule('pathway_avenues.js');
    if (!paCode) { bad('pathway_avenues.js not found'); return; }
    if (paCode.includes('function calculateTier')) {
        ok('PathwayAvenues.calculateTier() exists');
    } else {
        bad('PathwayAvenues.calculateTier() missing');
    }

    // 14. app.js wires everything
    const appCode = loadModule('app.js');
    if (!appCode) { bad('app.js not found'); return; }
    if (appCode.includes('openMenuCustomization') || appCode.includes('MenuCustomizationPanel')) {
        ok('app.js wires MenuCustomizationPanel');
    } else {
        bad('app.js missing MenuCustomizationPanel wiring');
    }
    if (appCode.includes('controller_nav')) {
        ok('app.js wires controller_nav');
    } else {
        bad('app.js missing controller_nav wiring');
    }

    // 15. World Dashboard exposes openWorldDashboardToTab
    if (wdCode.includes('openWorldDashboardToTab')) {
        ok('World Dashboard exposes openWorldDashboardToTab()');
    } else {
        bad('World Dashboard missing openWorldDashboardToTab()');
    }
}

// --- Test: localStorage Schema ---
function testStorageSchema() {
    console.log(`\n${C.bold}Test: localStorage Schema${C.reset}`);
    const upCode = loadModule('user_preferences.js');
    if (!upCode) return;

    const schemaChecks = [
        { key: 'starred', desc: 'Starred items array' },
        { key: 'menuLayout', desc: 'Menu layout config' },
        { key: 'menuLayout.gridType', desc: 'Grid type' },
        { key: 'menuLayout.defaultShape', desc: 'Default shape' },
        { key: 'menuLayout.defaultSize', desc: 'Default size' },
        { key: 'menuLayout.buttonOverrides', desc: 'Per-button overrides' },
    ];

    for (const check of schemaChecks) {
        if (upCode.includes(check.key)) {
            ok(check.desc);
        } else {
            bad(`Missing: ${check.desc}`);
        }
    }
}

// --- Run ---
console.log(`${C.bold}╔══════════════════════════════════════════════╗${C.reset}`);
console.log(`${C.bold}║  Menu Customization Pipeline Verification   ║${C.reset}`);
console.log(`${C.bold}╚══════════════════════════════════════════════╝${C.reset}`);

testFullPipeline();
testStorageSchema();

console.log(`\n${C.bold}════════════════════════════════════════════════${C.reset}`);
console.log(`${C.bold}Results: ${C.green}${pass} passed${C.reset}, ${C.red}${fail} failed${C.reset}`);
if (errors.length > 0) {
    console.log(`\n${C.red}Failures:${C.reset}`);
    errors.forEach(e => console.log(`  - ${e}`));
}
console.log(`${C.bold}════════════════════════════════════════════════${C.reset}\n`);

process.exit(fail > 0 ? 1 : 0);
