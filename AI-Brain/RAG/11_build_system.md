# 11. Build System, Deployment & CI/CD Deep-Dive

> **Generated**: 2026-09-03
> **Scope**: NFT-Seduction build pipeline evolution, deployment architecture, and build drift analysis
> **Sources**: Dockerfile, render.yaml, deploy-wasm.yml, package.json, go.mod, go.sum, all shell/PowerShell/batch scripts, all build logs

---

## 1. Build System Architecture Overview

### 1.1 Split-WASM Model

The project uses a **dual-binary architecture**:

| Component | Target | Output | Build Tag |
|-----------|--------|--------|-----------|
| Server | `linux/amd64` (native) | `server-bin` | *(none)* |
| Client | `js/wasm` | `Public/main.wasm` | *(none)* |
| Console | `windows/linux amd64` | `nft-seduction-console` | `console` |
| Mobile | `android/ios arm64` | `nft-seduction-mobile` | `console,mobile` |

### 1.2 Build Orchestration (package.json npm scripts)

```
clean → wasm:init → wasm:build → sass:build → server:build
```

| Script | Purpose | Mechanism |
|--------|---------|-----------|
| `clean` | Remove artifacts | Node.js `fs.rmSync` |
| `wasm:init` | Copy `wasm_exec.js` from GOROOT | Node.js `fs.copyFileSync` |
| `wasm:build` | Compile Go → WASM | `GOOS=js GOARCH=wasm go build` |
| `sass:build` | Compile SCSS → CSS | `sass --style=compressed` |
| `server:build` | Compile Go → native binary | `CGO_ENABLED=0 go build -ldflags="-w -s"` |
| `build` | Full pipeline | Chained `npm run` |
| `start` | Dev mode | `go run .` |

### 1.3 Go Module Configuration

```
module virtualbabestt
go 1.25.7
```

Key dependencies: go-algorand-sdk, go-ethereum, gorilla/websocket, prometheus, joho/godotenv.

---

## 2. Build Script Inventory

### 2.1 Shell Scripts (POSIX)

| Script | Target | Details |
|--------|--------|---------|
| `build_mobile.sh` | Android / iOS | `GOOS=android/ios GOARCH=arm64 -tags mobile` |
| `build_console.sh` | Windows / Linux | `GOOS=windows/linux GOARCH=amd64 -tags console` |
| `entrypoint.sh` | Docker | Creates `$DATA_DIR`, execs server-bin |
| `check_scripts.sh` | Diagnostic | Verifies `<script>` tags in index.html |

### 2.2 PowerShell Scripts (Windows)

| Script | Target | Details |
|--------|--------|---------|
| `build_mobile.ps1` | Android / iOS | Same as .sh, with `-ErrorActionPreference Stop` |
| `build_console.ps1` | Windows / Linux | Same as .sh, adds `.exe` extension |
| `launch_dev_server.ps1` | Local DEV | Isolated `devdata/`, PORT=8090, auto-opens browser |

### 2.3 Batch Files (Windows / AI-ML Pipeline)

| Script | Purpose |
|--------|---------|
| `setup_bot_pathway.bat` | Bot/Pet LLM corpus builder (wraps ornith harness) |
| `setup_custom_quant_ornith.bat` | GPU imatrix quantization pipeline |
| `start_zap_matrix_server.bat` | Launches llama-server with 96k context |

---

## 3. Docker Configuration

### 3.1 Dockerfile (Multi-Stage)

```dockerfile
# Stage 1: Builder
FROM golang:1.24-alpine3.20 AS builder
RUN apk add --no-cache nodejs npm git
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY package*.json ./
RUN npm install
COPY . .
RUN npm run build                    # ← full build pipeline

# Stage 2: Runtime
FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata
WORKDIR /app
COPY --from=builder /app/server-bin .
COPY --from=builder /app/Public ./Public
COPY --from=builder /app/entrypoint.sh .
COPY --from=builder /app/networks.json .
COPY --from=builder /app/season.json* .
ENV PORT=8088 DATA_DIR=/app/data NODE_ENV=production
RUN chmod +x ./entrypoint.sh ./server-bin
HEALTHCHECK --interval=30s --timeout=10s --retries=3 \
  CMD wget -qO- http://localhost:8088/api/health || exit 1
EXPOSE 8088
ENTRYPOINT ["./entrypoint.sh"]
CMD ["./server-bin"]
```

### 3.2 Docker Build Pipeline Steps

1. **Dependency Layer**: `go mod download` + `npm install` (cached unless changed)
2. **Source Ingestion**: Full `COPY . .`
3. **Build Execution**: `npm run build` (clean → wasm:init → wasm:build → sass:build → server:build)
4. **Artifact Transfer**: server-bin, Public/, entrypoint.sh, networks.json, season.json
5. **Runtime Hardening**: Alpine 3.20, ca-certificates, tzdata, healthcheck

---

## 4. Deployment Architecture

### 4.1 Render (Primary Platform)

**`render.yaml`** — Infrastructure-as-Code export (2026-05-18):

```yaml
version: "1"
projects:
  - name: VOiconomy faucet
    environments:
      - name: Deployment
        services:
          - type: web
            name: NFT-Seduction
            runtime: docker
            repo: https://github.com/slapkarnts/NFT-Seduction
            branch: Dev2
            plan: free
            region: virginia
            dockerContext: .
            dockerfilePath: ./Dockerfile
            autoDeployTrigger: commit
```

**Environment Variables (37 total, all `sync: false`)**:
- Blockchain: `ALGOD_URL_VOI`, `INDEXER_URL_VOI`, `ALGOD_URL_ALGO`, `INDEXER_URL_ALGO`
- Game Economy: `VAULT_ADDRESS`, `FAUCET_MNEMONIC`, `REWARD_ASSET_ID`, `COLLECTION_ID`, etc.
- Auth: `ADMIN_KEY`, `ADMIN_WALLETS`
- Frontend: `WC_PROJECT_ID`, chain IDs
- **Multi-chain expansion** (planned, unimplemented in code):
  - `FLOW_*`, `WAX_*`, `BITCOIN_NODE_URL`, `SOLANA_*`, `POLYGON_*`, `ETH_*`
  - `VBV_ASSET_ID`, `ALGOD_TOKEN_VOI`

### 4.2 GitHub Actions (CI/CD)

**`deploy-wasm.yml`** — Build & Deploy WASM Frontend:

```yaml
name: Build and Deploy WASM Frontend
on:
  push:
    branches: [ slapkarnts/Dev2 ]
jobs:
  build-and-deploy:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with: { go-version: '1.23' }    # ← MISMATCH
      - name: Compile WASM
        run: GOOS=js GOARCH=wasm go build -o Public/main.wasm main.go
      - name: Deploy to Branch
        uses: peaceiris/actions-gh-pages@v4
        with:
          github_token: ${{ secrets.GITHUB_TOKEN }}
          publish_branch: deploy
          publish_dir: .
          exclude_assets: '.github,.env.example,...'
      - name: Trigger Render Deployment
        run: echo "WASM Compiled and pushed to 'deploy' branch."
```

### 4.3 Alternative Platform Configs

| Config | Present | Notes |
|--------|---------|-------|
| `docker-compose.yml` | ❌ | Not found |
| `railway.toml` | ❌ | Not found |
| `fly.toml` | ❌ | Not found |
| `Procfile` | ❌ | Not found |
| `.github/workflows/` | ❌ | **deploy-wasm.yml exists but NOT in `.github/workflows/`** |

---

## 5. Build Drift Analysis — Where It Broke

### 5.1 Critical Version Mismatches

| Component | Declared | Actual/Required | Severity |
|-----------|----------|-----------------|----------|
| `go.mod` | `go 1.25.7` | — | — |
| `deploy-wasm.yml` | `go-version: '1.23'` | Needs 1.25.7 | **CRITICAL** |
| Dockerfile | `golang:1.24-alpine3.20` | Needs 1.25.x | **CRITICAL** |
| `actions/setup-go@v5` | 1.23 | Needs 1.25.7 | **HIGH** |

### 5.2 Massive Compilation Failure (natbuild logs)

**Affected files**: `natbuild.log`, `natbuild_def.log`, `natbuild_full.log` (all **IDENTICAL** — ~500 errors each)

**Error Categories**:

| Category | Count (approx) | Examples |
|----------|----------------|----------|
| Undefined methods on `*Lobby` | ~40+ | `TrackCareerXP`, `ResolveEnvoiName`, `verifyBuyInTransaction`, `checkMojoSurgeAchievementLocked`, `DisconnectClient`, `savePersistentCardCache`, `saveOnboardedWallets`, `getVerifiedCards`, `checkAssetOptIn`, `isPlayerAffiliatedWithClubLocked`, `indexerRequest`, `syncStatsFromBlockchain`, `isJusticeAligned`, `loadRegistrationsFromIndexer`, `executeGracefulShutdown`, `clientWallets`, `ethClient`, `mapChainToNetworkName`, `isWalletRegistered`, `GetCareerProgress`, `checkTreasuryRecoveryAchievementLocked`, `GhostTaxTotal`, `StagnationTaxTotal` |
| Undefined fields on `PlayerStats` | ~25+ | `Career`, `CareerTier`, `Credits`, `ActiveBuffs`, `TotalDividendClaimedMicro`, `LastActivity`, `Wanted`, `Name`, `VBVBalance`, `ID`, `SectorTiles` |
| Undefined fields on `CareerXP` | ~15+ | `Level`, `Tiers`, `RoleName`, `GetCareerTier`, `GetRumorFeeDiscount`, `GetJusticeMissionFeeDiscount` |
| Undefined fields on `TournamentState` | ~5+ | `BuyInAmount`, `Pot` |
| `JusticeMission` struct fields | ~50+ | `ID`, `Title`, `Description`, `RewardMicro`, `Target`, `Type` (all undefined) |
| `MatchHistory` fields | ~20+ | `UnderworldContractID`, `IsUnderworldContractSuccess` |
| Type mismatches | ~15+ | `PlayerStats` vs `*PlayerStats`, `uint64` vs `int`, struct vs nil |
| Missing imports | ~10+ | `fmt`, `log`, `sync`, `atomic` |
| Signature mismatches | ~10+ | `writeJSON` (extra status arg), `InterceptAndAudit` (wrong arg count), `EvaluateCrossCareerXP` (wrong return count) |
| Unused imports/variables | ~10+ | `encoding/json`, `crypto/sha256`, `encoding/binary`, etc. |

### 5.3 Silent Failures (Empty Logs)

| Log File | Status | Interpretation |
|----------|--------|----------------|
| `wasmbuild.log` | **EMPTY** (0 bytes) | WASM build never ran or failed before output |
| `natbuild_uncap.log` | **EMPTY** (0 bytes) | "uncap" variant never executed |
| `natbuild_norm.log` | **EMPTY** (0 bytes) | "norm" variant never executed |
| `natbuild_fresh.log` | Has errors (identical to others) | "fresh" variant ran but broke identically |

### 5.4 Misleading Success Indicators

| Log File | Content | Reality |
|----------|---------|---------|
| `build_check.log` | `EXIT=0` | Only checks HTML script tag presence — NOT a build verification |
| `_t3.log` | "EMBED GUARD", "done depth=1 j=75871" | Embedding/diff tool output, not build status |

### 5.5 Runtime Failures (server.log)

The server **starts** but operates in a degraded state:

```
[BOOTSTRAP WARNING] Local state recovery failed: bootstrap safety: attempt to initialize zero-value faucet
[CACHE ERROR] Indexer returned non-200 status for leaderboard snapshots: 404
[MultiChain] Algorand Mainnet health check failed: HTTP 401: Invalid API Token
[ORACLE WARNING] Node http://127.0.0.1:8082 failed gas check: HTTP 401: Invalid API Token
[ORACLE ERROR] Gas check failed: all cluster nodes degraded.
[SERVER HEALTH] Active Arena Matches: 0 | Vault Balance: 0.00 $VBV
```

**Runtime Issues**:
- Faucet balance: **0.00 $VBV** (no funds)
- All cluster nodes degraded (401 errors on local Algorand nodes)
- Indexer endpoints returning 404
- 0 AI citizens, 0 bonded assets, 0 model promotions
- Genesis boot detected (fresh state, no persisted data)

### 5.6 Go Vet Warnings (natbuild.log — vet phase)

```
economy_bootstrap.go:125: range var node copies lock: EntityMarketNode contains sync.RWMutex
ai_citizen_engine.go:509: log.Printf format %03d has arg of wrong type string
battle_service.go:730: fmt.Sprintf format %d has arg of wrong type float64
bonded_asset_registry.go:68: struct field has json tag but is not exported
server.go:1436: struct field Y/Z repeats json tag "x"
```

---

## 6. Build Pipeline Evolution Timeline

### Phase 1: Monolithic Go Server
- Single binary (`server.go` entrypoint)
- `go run .` / `go build .`
- Manual deployment

### Phase 2: Split-WASM Introduction
- Client-side WASM for browser execution
- `wasm:init` copies Go runtime's `wasm_exec.js`
- `wasm:build` with `GOOS=js GOARCH=wasm`
- npm scripts orchestrate multi-step pipeline

### Phase 3: Multi-Target Expansion
- Console builds (`-tags console`) for local authority sim
- Mobile builds (`-tags mobile`) for Android/iOS
- PowerShell scripts for Windows developers
- Isolated dev server (`launch_dev_server.ps1`)

### Phase 4: Containerization & Cloud Deployment
- Docker multi-stage build (golang:alpine → alpine)
- Render deployment (Docker runtime, free tier)
- GitHub Actions CI/CD (deploy-wasm.yml)
- Infrastructure-as-Code (render.yaml)
- 37 environment variables for multi-chain support

### Phase 5: AI/ML Integration (Parallel Track)
- ornith-matrix quantization pipeline
- Bot/Pet pathway local-LLM corpus builder
- llama-server orchestration
- **This track is separate from the main game build**

---

## 7. Root Cause Summary

### 7.1 Build Breaks Because:

1. **Codebase has grown faster than type definitions** — service files reference methods/fields on `Lobby`, `PlayerStats`, `CareerXP`, `TournamentState`, `JusticeMission`, and `MatchHistory` that either:
   - Were renamed/removed in a refactor
   - Exist in a different build-tag partition (`console` vs `mobile` vs server)
   - Were never implemented (planned features)

2. **Build tags are inconsistent** — services use `l.TrackCareerXP` etc. but the method is likely defined in a `_console.go` file that isn't compiled without the `console` tag

3. **Go version drift** — go.mod requires 1.25.7, Docker uses 1.24, CI uses 1.23

4. **WASM build is disconnected from main build** — `deploy-wasm.yml` builds WASM separately from the server, and the file is in the wrong location (not in `.github/workflows/`)

### 7.2 Deployment Breaks Because:

1. **No valid build artifact** — the natbuild logs show ~500 compilation errors, so `server-bin` likely doesn't exist or is stale
2. **Environment variables are placeholders** — `FAUCET_MNEMONIC`, `VAULT_ADDRESS`, `WC_PROJECT_ID` need real values
3. **Render free tier** spins down after inactivity — cold starts may timeout
4. **Local Algorand nodes unreachable** — 401 errors suggest the local node infrastructure is down or misconfigured
5. **Missing multi-chain config** — 12+ env vars for FLOW/WAX/SOLANA/POLYGON/ETH are declared but not implemented in code

### 7.3 The "Working" Illusion

- `build_check.log` shows `EXIT=0` — but this only checks HTML script tags, not compilation
- The server **starts** and binds to port 8088/8090 — but immediately reports zero balances and degraded nodes
- `natbuild_uncap.log` and `natbuild_norm.log` are empty — these build variants never completed
- The build pipeline **appears** to work (clean → init → build → sass → server) but each step may silently fail

---

## 8. Recommendations

1. **Immediate**: Fix Go version alignment (go.mod 1.25.7 → Dockerfile `golang:1.25.x` → CI `go-version: '1.25'`)
2. **Immediate**: Audit all undefined methods/fields — likely a build-tag partition issue
3. **Short-term**: Move `deploy-wasm.yml` to `.github/workflows/` or it won't execute
4. **Short-term**: Replace placeholder env vars in Render dashboard
5. **Medium-term**: Implement missing `JusticeMission` struct (50+ errors from this alone)
6. **Medium-term**: Unify `PlayerStats.Career`/`Credits`/`ActiveBuffs` field definitions
7. **Long-term**: Separate console/mobile/server builds into distinct packages with clear interfaces
8. **Long-term**: Add a real CI build verification step (not just `EXIT=0` from tag check)

---

## Appendix: File Inventory

| File | Type | Status |
|------|------|--------|
| `Dockerfile` | Docker build | ✅ Exists, Go version mismatch |
| `render.yaml` | Render IaC | ✅ Exists, 37 env vars |
| `deploy-wasm.yml` | GitHub Actions | ⚠️ Wrong location (not in `.github/workflows/`) |
| `package.json` | npm config | ✅ Exists, orchestrates build |
| `go.mod` | Go module | ✅ Exists, `go 1.25.7` |
| `go.sum` | Go checksums | ✅ Exists, 244 lines |
| `.env.example` | Config template | ✅ Exists, 82 lines |
| `entrypoint.sh` | Docker entry | ✅ Exists |
| `build_mobile.sh` | Shell script | ✅ Exists |
| `build_console.sh` | Shell script | ✅ Exists |
| `check_scripts.sh` | Shell script | ✅ Exists |
| `build_mobile.ps1` | PowerShell | ✅ Exists |
| `build_console.ps1` | PowerShell | ✅ Exists |
| `launch_dev_server.ps1` | PowerShell | ✅ Exists |
| `setup_bot_pathway.bat` | Batch | ✅ Exists |
| `setup_custom_quant_ornith.bat` | Batch | ✅ Exists |
| `start_zap_matrix_server.bat` | Batch | ✅ Exists |
| `build_check.log` | Log | ⚠️ Misleading EXIT=0 |
| `wasmbuild.log` | Log | ❌ Empty |
| `natbuild.log` | Log | ❌ ~500 errors |
| `natbuild_def.log` | Log | ❌ ~500 errors |
| `natbuild_full.log` | Log | ❌ ~500 errors |
| `natbuild_fresh.log` | Log | ❌ ~500 errors |
| `natbuild_uncap.log` | Log | ❌ Empty |
| `natbuild_norm.log` | Log | ❌ Empty |
| `server.log` | Log | ⚠️ Runtime degraded |
| `_t3.log` | Log | ℹ️ Diff tool output |
| `natvet.log` | Log | ⚠️ Go vet warnings |
