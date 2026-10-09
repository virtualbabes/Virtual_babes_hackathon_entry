Version: 4.1
Status: Current
Last Amended: 2026-09-20

# KEY 1 — SYNCHRONIZE

> Restore Repository Truth CHEAPLY, then stop reading. **The sweep IS the map** (506/506 files read; the result is on
> disk and held by gates). Re-deriving it from source is the most expensive mistake available in this repository.

---

# OBJECTIVE
Establish the true current state before any assessment, recommendation or implementation. Repository Truth overrides
conversational memory and stale documents. Repository-relative paths only.

---

# 0. FAST TRIAGE — cheapest signal first, in THIS order (do not start with reads)

**(a) Position — one command.**
`git rev-parse --abbrev-ref HEAD` · `git rev-parse --short HEAD` · `git status --porcelain` · the first
`**SESSION NOTE` lines of `.clinerules/Session-Handoff.md`.

**(b) The gates — seconds, no compile, and they are the repository's own opinion of itself.**
`verify:aspects` · `verify:routes-parity` · `verify:reachability` · `verify:routes` · `verify:duplicates` ·
`verify:handlers` · `verify:api`. **A FAIL is the session's FIRST FINDING**, not an obstacle: four of these exist
because a real defect hid behind a green reading. Never begin work on top of a failing gate.

**(c) Artifacts — 4 sizes + the wasm mtime**, to know whether the tree on disk is the tree that was reported.

**(d) Only now, reads.** If (a)–(c) agree with the handoff, go to KEY 2 with the position it names. If they disagree,
the handoff is stale: say so and reconcile before proceeding.

---

# 1. BINDING GOVERNANCE
1. `.clinerules/Directive-Protocol.md` — supreme authority for the KEY loop.
2. `.clinerules/active_directive.md` — the LIVE directive; replaced at KEY 3.5 if it predates the work.
3. `.clinerules/constitution.md` — no float on ledger paths (uint64 micro only) · no cloud models · AI citizens own
   wallets · contained overlays · secrets ENV-ONLY · **GIT PUSH is Brendan's**.
4. **`.clinerules/` IS DELIBERATELY LOCAL** (`.gitignore:8`; 0 tracked files) — Brendan's private method. **Never
   report it as a versioning gap and never "fix" the ignore rule**; `AI-Brain/**` is what travels with a push.

---

# 2. THE SWEEP — HOW TO READ IT WITHOUT RE-READING THE REPOSITORY
- **Start at the ROW, not the file:** `AI-Brain/RAG/17_aspects_flow.md` — 39 aspect rows (owner files · routes ·
  state · flow · UI · measured gaps · status) + a 39-row COVERAGE table. Most questions end here.
- **Then a per-file entry, only if needed:** `14_flow_go.md` · `15_flow_js.md` · `16_flow_css.md` · `18_mic_flow.md`.
  Find an entry by its heading; `19_coverage_ledger.md` is the index of what was read (treatment + ranges).
- `AI-Brain/App-Aspect-Index.md` — the 39 aspects and the **Player View** (what the app offers a user).
- **Display limits are real:** a read caps at ~2000 lines / ~47k chars and TRUNCATES THE MIDDLE of a multi-entry read.
  Read in contiguous ~180-line windows.
- **Never read the historical memory in full** (`Archived_A.I_memory.md`, 2758 lines): search it, or read its last 50.

---

# 3. ACTIVE STATE
- `.clinerules/Session-Handoff.md` — position, YOLO state, last session's traps. LOCAL.
- `AI-Brain/ToDo.md` — plans (the sweep's plan block sits at the end) · `AI-Brain/Problems.md` — actionable defects.
- `AI-Brain/MASTER-PLAN.md` — reconciled, but **its §6 rows are known stale: the aspect rows outrank it.**
- `AI-Brain/archive/docs-2026-09-07/A.I_memory.md` — the CURRENT memory file (small).
- `AI-Brain/NFT-Seduction-Absolute-Visoin.md` — vision. Never edit.

---

# 4. BUILD BASELINE — only if this session will change Go (otherwise (c) is enough)
A bare `go build ./...` EXIT 0 is a **FALSE GREEN** for the server: the shell default compiles the client only and
excludes every `//go:build !js && !wasm` file. Establish the four targets, **ONE BUILD PER COMMAND** (the shell caps at
~30s; a batch of builds times out and reads as failure), each with `-o` to TEMP — never into the tree, where a bare
`go build` drops a binary named after the directory:
native · `GOOS=linux GOARCH=amd64` · `GOOS=js GOARCH=wasm` · `-tags console`.
Then `go test -count=1 .`: **exactly ONE failure is expected — the recorded pre-existing AMM test.** Any other failure
is a regression and is the first thing to report. Prove `Public/main.wasm` untouched by **size AND mtime**.

---

# 5. COMMAND HYGIENE (each rule here has cost a session)
- A native command's **stderr terminates the pipeline** in PowerShell: use `cmd /c "… 2>&1"` or `2> file`, then read it.
- `npm.ps1` is policy-blocked → use **`npm.cmd`**; a `.ps1` script needs `-ExecutionPolicy Bypass -File`.
- **One build per command; one write per file per batch** — and never verify a write in the same batch as the write.
- Read with an explicit UTF8 reader, write with `UTF8Encoding($false)`; never put non-ASCII literals in a shell string.

---

# VERIFY THE POSITION YOU WERE HANDED (accuracy gate)
Does the branch/HEAD match the handoff? Do the artifact sizes/mtimes match? Did every gate pass? If any answer is no,
the handoff is stale — **say so explicitly and reconcile**, because every later decision inherits it. A position report
is only true if it was MEASURED this session.

# REPORT
Position (branch · HEAD · changed paths) · gate results, one line each · build baseline (or why it was skipped) ·
repository health · active priorities · blockers · Repository Truth status · what is NOT established.
If Repository Truth cannot be established: **stop and explain why.**

# TRANSITION
Proceed to **KEY 2 — ASSESS**. Do not recommend and do not implement.
