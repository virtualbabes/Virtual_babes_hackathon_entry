Version: 2.1
Status: Stable
Last Amended: 2026-09-04

# Documentation

> **Knowledge organization for NFT-Seduction.**
>
> Documentation supports the repository. Repository Truth prevails.

---

# PRINCIPLES

The repository is the authoritative implementation.

Documentation:

* Explains implementation
* Preserves architectural knowledge
* Enables consistent development
* Reduces context recovery

Keep documentation concise, current and synchronized.

---

# CONSTITUTION
* Work Space Directory folder location:
`"Z:\Crypto_Draught\NFT-Seduction"`
Ignorelist
* `.clineignore`
* **`.clinerules/` IS DELIBERATELY LOCAL — operator decision, 2026-09-20.** `.gitignore:8` excludes it, so
  **0 files under `.clinerules` are tracked by git**: the five Keys, this document, the constitution, the active
  directive and `Session-Handoff.md` live on this machine **BY DESIGN** — in Brendan's words, *"it is my secret way
  to build apps"*. **Do not report it as a versioning gap, do not "fix" the ignore rule, and never move a rule
  document out of `.clinerules` to make it trackable.** `AI-Brain/**` IS tracked and travels with a push.

Read in order:

1. `Z:\Crypto_Draught\NFT-Seduction\.clinerules\lead-systems-architect.md`
2. `Z:\Crypto_Draught\NFT-Seduction\.clinerules\vision-philosophy.md`
3. `Z:\Crypto_Draught\NFT-Seduction\.clinerules\repository-truth.md`
4. `Z:\Crypto_Draught\NFT-Seduction\.clinerules\system-protocol.md`
5. `Z:\Crypto_Draught\NFT-Seduction\.clinerules\tooling-integrity.md`
6. `Z:\Crypto_Draught\NFT-Seduction\.clinerules\architecture-ledger.md`
7. `Z:\Crypto_Draught\NFT-Seduction\.clinerules\persistence.md`
8. `Z:\Crypto_Draught\NFT-Seduction\.clinerules\Documentation.md`
9. `Z:\Crypto_Draught\NFT-Seduction\.clinerules\browser-safety.md`

---

# ACTIVE DOCUMENTS

### Vision

* `Z:\Crypto_Draught\NFT-Seduction\AI-Brain\NFT-Seduction-Absolute-Visoin.md`

### Repository State

* `Z:\Crypto_Draught\NFT-Seduction\.clinerules\Session-Handoff.md` 
* `Z:\Crypto_Draught\NFT-Seduction\AI-Brain\ToDo.md`
* `Z:\Crypto_Draught\NFT-Seduction\AI-Brain\Problems.md`
* `Z:\Crypto_Draught\NFT-Seduction\AI-Brain\Docbase-Analysis.md (DOES NOT EXIST - verified 2026-09-20; superseded by AI-Brain/App-Aspect-Index.md and the RAG aspect rows)` — **DOES NOT EXIST (verified 2026-09-20).** This line named a file that is not on disk; the document map is corrected here rather than left as a false statement. What replaces it as the reconciled state owner: **`AI-Brain/App-Aspect-Index.md`** (39 aspects) and the sweep's rows in **`AI-Brain/RAG/17_aspects_flow.md`**.
* `Z:\Crypto_Draught\NFT-Seduction\AI-Brain\archive\docs-2026-09-07\A.I_memory.md`

### Architecture Owners (single source of truth per domain)

* `Z:\Crypto_Draught\NFT-Seduction\AI-Brain\Rivalry-Matrix.md` — **the rivalry architecture**: the
  three matrices (path / career / region), their weights, the career-path choice, the civil rank,
  the promotion and demotion lifecycle, and the registered endpoint list. Every other document that
  mentions rivalry is a VIEW of this file; where they disagree, this file wins.

### Repository Structure

* `Z:\Crypto_Draught\NFT-Seduction\AI-Brain\File-Flow-Overview-1.md`

### The Sweep's Flow Corpus (2026-09-20 — the owner of the derived aspect rows)

* `Z:\Crypto_Draught\NFT-Seduction\AI-Brain\RAG\14_flow_go.md` — every Go file, per-file evidence.
* `Z:\Crypto_Draught\NFT-Seduction\AI-Brain\RAG\15_flow_js.md` — every first-party JS file (169).
* `Z:\Crypto_Draught\NFT-Seduction\AI-Brain\RAG\16_flow_css.md` — every SCSS partial and the compiled stylesheet (110).
* `Z:\Crypto_Draught\NFT-Seduction\AI-Brain\RAG\18_mic_flow.md` — data, config, CI, tooling and vendor provenance (73).
* `Z:\Crypto_Draught\NFT-Seduction\AI-Brain\RAG\17_aspects_flow.md` — **the derived ASPECT ROWS (39/39)** plus the ASPECT COVERAGE table; `npm run verify:aspects` holds it closed.
* `Z:\Crypto_Draught\NFT-Seduction\AI-Brain\RAG\19_coverage_ledger.md` — the read census: 506 files, each with its treatment and read ranges (`506 read / 0 partial / 0 pending`).

**The sweep's gates** — each fails CLOSED and each carries a `--selftest` proving it can report a failure:
`npm run verify:aspects` (every numbered aspect of the index carries a row, and no mapping is stale) and
`npm run verify:routes-parity` (the two entrypoints' route tables cannot drift: a new unbaselined path, a
stale exemption, or a parser that finds nothing all fail the build).

### Repository State

### Institutional Memory
Current memory file to retain session implementation - CORRECTED 2026-09-20: this line and two others named a memory file that DOES NOT EXIST, and named the 2758-line HISTORICAL archive as the current one. What is on disk (Get-ChildItem AI-Brain -Recurse -Filter '*memory*') is exactly two files:
* `AI-Brain\archive\docs-2026-09-07\A.I_memory.md` - THE CURRENT MEMORY FILE (small; extend it each session, as this pass did).
* `AI-Brain\archive\docs-2026-09-07\Archived_A.I_memory.md` - the HISTORICAL archive (2758 lines): SEARCH it, never read it in full, extend only for significant architectural knowledge.
### Historical Memory
* [search this file, dont read this file in full it is 2700+ lines currently]
 `Z:\Crypto_Draught\NFT-Seduction\AI-Brain\archive\docs-2026-09-07\Archived_A.I_memory.md`

Update only for significant architectural knowledge, reading the end 50 lines of the file only pre-update.

Not for routine session continuity.

---

# DOCUMENT OWNERSHIP

| Document              | Responsibility          |
| --------------------- | ----------------------- |
| Vision Philosophy     | Design philosophy       |
| Repository Truth      | Source of truth         |
| System Protocol       | Operational workflow    |
| Tooling Integrity     | Verification            |
| Architecture Ledger   | Engineering constraints |
| Browser Safety        | Browsing Rules          |
| Persistence           | Recovery & durability   |
| Documentation         | Knowledge organization  |
| Session-Handoff       | Active session state    |
| A.I_memory (CURRENT - the small one)            | Institutional knowledge |
| Archived_A.I_memory.md (HISTORICAL - 2758 lines) | Historical Knowledge    |

* Short term work documents {`Z:\Crypto_Draught\NFT-Seduction\AI-Brain\archive\docs-2026-09-07\A.I_memory.md`
`Z:\Crypto_Draught\NFT-Seduction\AI-Brain\ToDo.md`
`Z:\Crypto_Draught\NFT-Seduction\.clinerules\Session-Handoff.md`
`Z:\Crypto_Draught\NFT-Seduction\AI-Brain\Problems.md`
`Z:\Crypto_Draught\NFT-Seduction\AI-Brain\archive\docs-2026-09-07\orphan_analysis.md` (ARCHIVED 2026-09-07 — a pre-refactor reconciliation, historical only)}
* Long-term Documents {`Z:\Crypto_Draught\NFT-Seduction\AI-Brain\Docbase-Analysis.md (DOES NOT EXIST - verified 2026-09-20; superseded by AI-Brain/App-Aspect-Index.md and the RAG aspect rows)`
`Z:\Crypto_Draught\NFT-Seduction\AI-Brain\File-Flow-Overview-1.md`
`Z:\Crypto_Draught\NFT-Seduction\AI-Brain\archive\docs-2026-09-07\orphan_fix_list.md` (ARCHIVED 2026-09-07 — historical only)}

One document.

One responsibility.

No overlap.

---

# MAINTENANCE

Prefer updating existing documentation over creating new documents.

Remove obsolete information.

Keep documentation synchronized with implementation.

---

# FINAL PRINCIPLE

Good documentation explains the repository.

Great documentation minimizes context recovery.
