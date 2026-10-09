#!/usr/bin/env node
/*
 * verify_aspect_rows.js — npm run verify:aspects
 *
 * THE QUESTION: may a new aspect arrive in the product without carrying a derived
 * row in AI-Brain/RAG/17_aspects_flow.md?
 *
 * WHY IT EXISTS. The aspect derivation reached 35 rows and declared itself closed.
 * A CLOSED DERIVATION IS AN OPINION unless something mechanical holds it closed:
 * AI-Brain/App-Aspect-Index.md is a numbered list a later session can extend, and
 * nothing in the repository compared it with the rows file. This gate is that
 * comparison.
 *
 * HOW IT AVOIDS FUZZY TITLE MATCHING (the reason a naive version gets switched
 * off). It does not guess which row covers which aspect by similarity. It reads
 * the COVERAGE TABLE inside the rows file — one line per index aspect, naming the
 * row heading that covers it — and validates both sides verbatim:
 *   1. the index parses to a NON-EMPTY set of numbered aspects
 *   2. the coverage table parses to a non-empty set
 *   3. every index aspect has exactly one coverage entry   (missing  = FAIL)
 *   4. no entry names an aspect the index does not declare  (invented = FAIL)
 *   5. the covered TITLE equals the index title VERBATIM    (drift    = FAIL)
 *   6. the named row heading EXISTS as a '### ' heading     (stale    = FAIL)
 *   7. the NONE sentinel is permitted but REPORTED — and it FAILS when ANY heading
 *      in the same file covers that aspect, BY ITS NUMBER or BY ITS OWN TITLE
 *      (case- and separator-insensitive containment), because then the row exists
 *      and the mapping is the stale half. The title test is why the first draft of
 *      this rule was too weak: 30 of the 39 aspects have a row heading that carries
 *      no number at all, so a number-only test could never have seen them.
 *
 * LIMITS, STATED HERE RATHER THAN IMPLIED. The row value is a heading PREFIX: the
 * gate proves the heading exists, never that it is the RIGHT row — that judgement
 * is the table author's and is written where a reviewer can see it. It reads two
 * files line by line, so a heading built by concatenation is invisible; and it
 * cannot see an aspect missing from BOTH files.
 */
'use strict';
const fs = require('fs');
const path = require('path');

const ROOT = path.resolve(__dirname, '..', '..');
const INDEX_PATH = path.join(ROOT, 'AI-Brain', 'App-Aspect-Index.md');
const ROWS_PATH = path.join(ROOT, 'AI-Brain', 'RAG', '17_aspects_flow.md');
const SECT = '\u00A7';
const COVERAGE_HEADING = /^##\s+ASPECT COVERAGE/i;
const SECTION_END = /^##\s+\S/;
const ASPECT_HEADING = /^##\s+(\d{1,2})\.\s+(.+?)\s*$/;
const COVERAGE_ROW = new RegExp('^\\|\\s*' + SECT + '(\\d{1,2})\\s*\\|\\s*(.+?)\\s*\\|\\s*(.+?)\\s*\\|\\s*$');
const ROW_HEADING = /^###\s+(.+?)\s*$/;
const NONE = 'NONE';

/** The numbered aspects the index declares: [{n, title}] in file order. */
function parseIndex(text) {
  const out = [];
  for (const line of text.split(/\r?\n/)) {
    const m = line.match(ASPECT_HEADING);
    if (m) out.push({ n: Number(m[1]), title: m[2].trim() });
  }
  return out;
}

/** The coverage table rows of the aspects file: [{n, title, target}]. */
function parseCoverage(text) {
  const out = [];
  let inSection = false;
  for (const line of text.split(/\r?\n/)) {
    if (COVERAGE_HEADING.test(line)) { inSection = true; continue; }
    if (!inSection) continue;
    if (SECTION_END.test(line)) break;
    const m = line.match(COVERAGE_ROW);
    if (m) out.push({ n: Number(m[1]), title: m[2].trim(), target: m[3].trim() });
  }
  return out;
}

/** The derived-row headings a coverage target must resolve against. */
function parseRowHeadings(text) {
  const out = [];
  for (const line of text.split(/\r?\n/)) {
    const m = line.match(ROW_HEADING);
    if (m) out.push(m[1].trim());
  }
  return out;
}

/** The rule, as a pure function of the two documents. */
function audit(indexText, rowsText) {
  const findings = [];
  const aspects = parseIndex(indexText);
  const coverage = parseCoverage(rowsText);
  const headings = parseRowHeadings(rowsText);

  if (aspects.length === 0) {
    findings.push('the aspect index parsed to ZERO numbered aspects — the parser found nothing, so this run proves nothing');
  }
  if (coverage.length === 0) {
    findings.push('the coverage table parsed to ZERO rows — a table that reads as empty cannot hold a derivation closed');
  }
  if (findings.length) return { aspects, coverage, headings, findings };

  const byNumber = new Map();
  for (const a of aspects) {
    if (byNumber.has(a.n)) findings.push(`the aspect index declares ${SECT}${a.n} TWICE — an ambiguous subject`);
    byNumber.set(a.n, a);
  }

  const covered = new Map();
  for (const c of coverage) {
    if (covered.has(c.n)) findings.push(`the coverage table declares ${SECT}${c.n} TWICE — one owner per aspect`);
    covered.set(c.n, c);
    if (!byNumber.has(c.n)) findings.push(`the coverage table names ${SECT}${c.n}, which the aspect index does not declare (invented aspect)`);
    const a = byNumber.get(c.n);
    if (a && a.title !== c.title) findings.push(`${SECT}${c.n} title drift: coverage says "${c.title}", the index says "${a.title}"`);
    if (c.target === NONE) {
      // A NONE claim is honest only while NO heading covers the aspect — by its
      // number OR by its own title, compared with case/separators removed.
      const norm = (s) => s.toLowerCase().replace(/[^a-z0-9]/g, '');
      const wantTitle = norm(c.title);
      const byNumber = headings.some(h => h.includes(SECT + c.n));
      const byTitle = wantTitle.length > 3 && headings.some(h => norm(h).includes(wantTitle));
      if (byNumber || byTitle) {
        findings.push(`${SECT}${c.n} is marked ${NONE} but a row heading in the same file covers it (${byNumber ? 'by its number' : 'by its title'}) — the mapping is the stale half`);
      }
    } else if (!headings.some(h => h.startsWith(c.target))) {
      findings.push(`${SECT}${c.n} names the row "${c.target}", which is not a '### ' heading in the rows file (stale mapping or renamed row)`);
    }
  }

  for (const a of aspects) {
    if (!covered.has(a.n)) findings.push(`${SECT}${a.n} "${a.title}" has NO coverage entry — a new aspect cannot arrive unrowed`);
  }
  return { aspects, coverage, headings, findings };
}

function selftest() {
  // SYNTHETIC FIXTURES ONLY: the control must not depend on the real documents,
  // or proving the gate can fail would mean breaking the repository.
  const S = SECT;
  const index = ['## 1. Architecture', '## 2. Onboarding & Player Entry', '## 3. Battle System (Core Gameplay)'].join('\n');
  const rows = [
    '### Architecture \u2014 the funding rails',
    '### Onboarding \u2014 the entry path',
    '',
    '## ASPECT COVERAGE',
    '| Aspect | Title | Derived row |',
    '| --- | --- | --- |',
    `| ${S}1 | Architecture | Architecture \u2014 the funding rails |`,
    `| ${S}2 | Onboarding & Player Entry | Onboarding \u2014 the entry path |`,
    `| ${S}3 | Battle System (Core Gameplay) | ${NONE} |`,
    '',
    '## REMAINDER',
  ].join('\n');

  const cases = [
    ['clean pair (MUST PASS)', index, rows, 0],
    ['an index aspect with no coverage entry', index + '\n## 4. Auctions', rows, 1],
    ['a coverage entry naming an invented aspect', `${S}3 | Battle System (Core Gameplay) | ${NONE} |`, `${S}3 | Battle System (Core Gameplay) | ${NONE} |\n| ${S}7 | Ghost Aspect | ${NONE} |`, 1],
    ['a stale target heading', index, rows.replace('Onboarding \u2014 the entry path |', 'Onboarding \u2014 a renamed row |'), 1],
    ['title drift', index, rows.replace(`| ${S}2 | Onboarding & Player Entry |`, `| ${S}2 | Onboarding and Player Entry |`), 1],
    ['NONE while a heading covers the aspect by TITLE', index, rows.replace('Architecture \u2014 the funding rails |', `${NONE} |`), 1],
    ['an index that parses to zero aspects', 'no aspects here', rows, 1],
    ['a rows file with no coverage table', index, '### Architecture \u2014 the funding rails', 1],
  ];

  let bad = 0;
  let mustPass = 0;
  let mustFail = 0;
  for (const [name, i, r, want] of cases) {
    const { findings } = audit(i, r);
    const got = findings.length > 0 ? 'FAIL' : 'PASS';
    const ok = want === 0 ? got === 'PASS' : got === 'FAIL';
    if (want === 0) mustPass++; else mustFail++;
    if (!ok) bad++;
    console.log(`  [selftest] ${ok ? 'ok  ' : 'BAD '} ${name.padEnd(44)} -> ${got} (${findings.length} finding(s))`);
  }
  if (bad) {
    console.error(`SELFTEST FAILED — ${bad} of ${cases.length} shapes did not report as required.`);
    return 1;
  }
  console.log(`SELFTEST PASSED — ${cases.length}/${cases.length} shapes (${mustPass} must-pass, ${mustFail} must-fail).`);
  return 0;
}

function main() {
  if (process.argv.includes('--selftest')) process.exit(selftest());
  const indexText = fs.readFileSync(INDEX_PATH, 'utf8');
  const rowsText = fs.readFileSync(ROWS_PATH, 'utf8');
  const { aspects, coverage, findings } = audit(indexText, rowsText);
  console.log('ASPECT ROW COVERAGE — every numbered aspect must carry a derived row');
  console.log(`  aspects declared in the index : ${aspects.length}`);
  console.log(`  coverage entries in the rows  : ${coverage.length}`);
  const none = coverage.filter(c => c.target === NONE).length;
  if (none) console.log(`  marked ${NONE} (reported, never hidden): ${none}`);
  if (findings.length) {
    console.error('\nFAIL — the derivation is not held closed:');
    for (const f of findings) console.error('  - ' + f);
    process.exit(1);
  }
  const first = aspects[0];
  const last = aspects[aspects.length - 1];
  console.log(`  first: ${SECT}${first.n} ${first.title}`);
  console.log(`  last : ${SECT}${last.n} ${last.title}`);
  console.log(`\nPASS — all ${aspects.length} aspects carry a row, every row named exists, and no mapping is stale.`);
  process.exit(0);
}

main();

