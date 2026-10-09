#!/usr/bin/env node
// ============================================================================
// flow_debug.js — Menu flow + controller nav debugger
// ============================================================================

const fs = require('fs');
const path = require('path');

const PROJECT_ROOT = path.resolve(__dirname, '../..');
const JS_ROOT = path.join(PROJECT_ROOT, 'Public', 'js');
const PUBLIC_ROOT = path.join(PROJECT_ROOT, 'Public');

const C = { reset: '\x1b[0m', green: '\x1b[32m', red: '\x1b[31m', yellow: '\x1b[33m', cyan: '\x1b[36m', bold: '\x1b[1m' };
let pass = 0, fail = 0;
const errors = [];

function ok(msg) { console.log(`  ${C.green}✓${C.reset} ${msg}`); pass++; }
function bad(msg) { console.log(`  ${C.red}✗${C.reset} ${msg}`); fail++; errors.push(msg); }
function info(msg) { console.log(`  ${C.cyan}→${C.reset} ${msg}`); }

function loadModule(f) {
    const p1 = path.join(JS_ROOT, f);
    const p2 = path.join(PUBLIC_ROOT, f);
    return fs.existsSync(p1) ? fs.readFileSync(p1, 'utf8') : (fs.existsSync(p2) ? fs.readFileSync(p2, 'utf8') : null);
}

// Parse top-level keys from an object literal: const FOO = { ... };
function extractObjectKeys(code, name) {
    const re = new RegExp(`const ${name} = \\{([\\s\\S]*?)\\n\\};`);
    const m = code.match(re);
    if (!m) return [];
    const body = m[1];
    // Match top-level keys (lines starting with optional whitespace + word + colon)
    const keys = new Set();
    const keyRe = /^\s{4}(\w+):/gm;
    let km;
    while ((km = keyRe.exec(body)) !== null) {
        keys.add(km[1]);
    }
    return [...keys];
}

function extractObjectBody(code, name) {
    const re = new RegExp(`const ${name} = \\{([\\s\\S]*?)\\n\\};`);
    const m = code.match(re);
    return m ? m[1] : '';
}

// --- Test 1 ---
function testShapes() {
    console.log(`\n${C.bold}Test 1: Shape Library${C.reset}`);
    const code = loadModule('menu_customization.js');
    if (!code) { bad('menu_customization.js not found'); return; }

    const shapeIds = extractObjectKeys(code, 'SHAPES');
    info(`Found ${shapeIds.length} shapes: ${shapeIds.slice(0, 8).join(', ')}...`);

    let valid = 0;
    for (const id of shapeIds) {
        const re = new RegExp(`${id}:\\s*\\{[\\s\\S]*?clip:\\s*['"\`]([^'"\`]+)['"\`]`);
        if (re.test(code)) valid++; else bad(`Shape "${id}" has invalid clip-path`);
    }
    ok(`${valid}/${shapeIds.length} shapes have valid clip-path`);

    const pw = extractObjectBody(code, 'PATHWAY_SHAPES');
    if (pw.length > 0) ok(`${(pw.match(/P-/g) || []).length} pathway shapes mapped`); else bad('PATHWAY_SHAPES missing');

    const sw = extractObjectBody(code, 'SYSTEM_SHAPES');
    if (sw.length > 0) ok(`${(sw.match(/(?<=')\w+(?=':)/g) || []).length} system shapes mapped`); else bad('SYSTEM_SHAPES missing');
}

// --- Test 2 ---
function testSizes() {
    console.log(`\n${C.bold}Test 2: Size Tiers${C.reset}`);
    const code = loadModule('menu_customization.js');
    if (!code) { bad('menu_customization.js not found'); return; }

    const ids = extractObjectKeys(code, 'SIZES');
    ok(`${ids.length} size tiers: ${ids.join(', ')}`);
    for (const id of ids) {
        const re = new RegExp(`${id}:\\s*\\{[\\s\\S]*?px:\\s*(\\d+),[\\s\\S]*?font:\\s*(\\d+),[\\s\\S]*?icon:\\s*(\\d+)`);
        const m = code.match(re);
        if (m) ok(`Size "${id}": ${m[1]}px, font ${m[2]}px, icon ${m[3]}px`);
        else bad(`Size "${id}" missing properties`);
    }
}

// --- Test 3 ---
function testGrids() {
    console.log(`\n${C.bold}Test 3: Grid Layouts${C.reset}`);
    const code = loadModule('menu_customization.js');
    if (!code) { bad('menu_customization.js not found'); return; }

    const ids = extractObjectKeys(code, 'GRIDS');
    ok(`${ids.length} grid types: ${ids.join(', ')}`);

    if (code.includes('function computePositions') || code.includes('computePositions = function') || code.includes('computePositions(')) {
        ok('computePositions() function found');
    } else {
        bad('computePositions() function missing');
    }

    let navCount = 0;
    for (const id of ids) {
        const re = new RegExp(`${id}:\\s*\\{[\\s\\S]*?navType:\\s*['"\`]([^'"\`]+)['"\`]`);
        if (re.test(code)) { ok(`Grid "${id}" has navType`); navCount++; }
        else bad(`Grid "${id}" missing navType`);
    }
}

// --- Test 4 ---
function testUserPreferences() {
    console.log(`\n${C.bold}Test 4: UserPreferences${C.reset}`);
    const code = loadModule('user_preferences.js');
    if (!code) { bad('user_preferences.js not found'); return; }

    if (code.includes('menuLayout')) ok('menuLayout in DEFAULTS'); else bad('menuLayout missing from DEFAULTS');

    const funcs = ['getMenuLayout','setMenuLayout','getButtonOverride','setButtonOverride','removeButtonOverride','getDefaultShape','getDefaultSize','setDefaultShape','setDefaultSize','getGridType','setGridType'];
    for (const f of funcs) {
        if (code.includes(`function ${f}`)) ok(`Function ${f}()`); else bad(`Function ${f}() missing`);
    }

    const expose = code.match(/window\.UserPreferences = \{([\s\S]*?)\n\};/);
    if (expose) {
        for (const f of ['getMenuLayout','getButtonOverride','getDefaultShape']) {
            if (expose[1].includes(f)) ok(`UserPreferences.${f} exposed`); else bad(`UserPreferences.${f} not exposed`);
        }
    }
}

// --- Test 5 ---
function testControllerNav() {
    console.log(`\n${C.bold}Test 5: Controller Nav${C.reset}`);
    const code = loadModule('controller_nav.js');
    if (!code) { bad('controller_nav.js not found'); return; }

    if (code.includes('import { getControllerNav }')) ok('Imports getControllerNav'); else bad('Missing getControllerNav import');

    const grids = ['grid','circle','linear-h','linear-v','cross','diamond','arc','triangle'];
    let handled = 0;
    for (const g of grids) {
        if (code.includes(`case '${g}'`)) handled++; else bad(`Missing handler for "${g}"`);
    }
    ok(`${handled}/${grids.length} grid types handled`);

    if (code.includes('function moveFocus')) ok('moveFocus() found'); else bad('moveFocus() missing');
    if (code.includes('pollGamepad')) ok('pollGamepad() found'); else bad('pollGamepad() missing');
}

// --- Test 6 ---
function testRenderStarred() {
    console.log(`\n${C.bold}Test 6: renderStarredItems${C.reset}`);
    const code = loadModule('menu-constellation.js');
    if (!code) { bad('menu-constellation.js not found'); return; }

    const checks = [
        ['computePositions', 'Calls computePositions()'],
        ['getClipPath', 'Calls getClipPath()'],
        ['getMenuLayout', 'Reads getMenuLayout()'],
        ['position: absolute', 'Supports absolute positioning'],
        ['clip-path', 'Applies clip-path'],
        ['size-', 'Applies size classes'],
    ];
    for (const [needle, desc] of checks) {
        if (code.includes(needle)) ok(desc); else bad(`Missing: ${desc}`);
    }
}

// --- Test 7 ---
function testPathways() {
    console.log(`\n${C.bold}Test 7: Pathway Avenues${C.reset}`);
    const code = loadModule('pathway_avenues.js');
    if (!code) { bad('pathway_avenues.js not found'); return; }

    for (const f of ['calculateTier','getTierProgress','getCSSVariables','injectAnimationStyles']) {
        if (code.includes(`function ${f}`)) ok(`Function ${f}()`); else bad(`Function ${f}() missing`);
    }
    const body = extractObjectBody(code, 'PATHWAYS');
    const count = (body.match(/P-/g) || []).length;
    ok(`${count} pathways defined`);
}

// --- Test 8 ---
function testAppJs() {
    console.log(`\n${C.bold}Test 8: app.js Integration${C.reset}`);
    const code = loadModule('app.js') || loadModule('../app.js');
    if (!code) { bad('app.js not found'); return; }

    for (const imp of ['menu_customization','pathway_avenues','user_preferences','menu_customization_panel','controller_nav']) {
        if (code.includes(imp)) ok(`Imports ${imp}.js`); else bad(`Missing import: ${imp}.js`);
    }
}

// --- Run ---
console.log(`${C.bold}╔══════════════════════════════════════════════╗${C.reset}`);
console.log(`${C.bold}║  Menu Flow + Controller Nav Debugger        ║${C.reset}`);
console.log(`${C.bold}╚══════════════════════════════════════════════╝${C.reset}`);

testShapes(); testSizes(); testGrids(); testUserPreferences(); testControllerNav(); testRenderStarred(); testPathways(); testAppJs();

console.log(`\n${C.bold}════════════════════════════════════════════════${C.reset}`);
console.log(`${C.bold}Results: ${C.green}${pass} passed${C.reset}, ${C.red}${fail} failed${C.reset}`);
if (errors.length) { console.log(`\n${C.red}Failures:${C.reset}`); errors.forEach(e => console.log(`  - ${e}`)); }
console.log(`${C.bold}════════════════════════════════════════════════${C.reset}\n`);

process.exit(fail > 0 ? 1 : 0);
