# dev_server.ps1 — Unified local DEV server for NFT-Seduction (Lead Architect plug-in)
# Consolidates launch_dev_server.ps1 + dev_watcher.ps1 into ONE tool implementing the
# .clinerules/dev_server_automation spec: isolated state (./devdata), .env seed, optional
# WASM+SASS+server build, auto-open browser+DevTools, file-watcher (Go->rebuild/restart,
# SCSS->sass, JS->refresh), health watchdog w/ auto-restart, multi-process manager, and
# optional -Test CDP UI/function smoke-test harness.
#
# Usage:
#   .\dev_server.ps1                 # build server+sass, watch, open browser+devtools
#   .\dev_server.ps1 -Prebuilt       # skip build, use ./server-bin(.exe)
#   .\dev_server.ps1 -FullBuild      # also (re)build Public/main.wasm
#   .\dev_server.ps1 -NoOpen         # don't auto-open browser
#   .\dev_server.ps1 -NoWatch        # foreground server, no watcher/health loop
#   .\dev_server.ps1 -Test           # after healthy, run the CDP UI/function harness

param(
    [switch]$Prebuilt,
    [switch]$FullBuild,
    [switch]$NoOpen,
    [switch]$NoWatch,
    [switch]$Test,
    [int]$Port = 8090,
    [string]$DataDir = ".\devdata",
    [int]$CdpPort = 9222
)

$ErrorActionPreference = "Stop"
$RepoRoot = Split-Path -Parent (Split-Path -Parent $PSScriptRoot)   # $PSScriptRoot = .../tools/server -> repo root is two levels up
Push-Location $RepoRoot

# npm on Windows resolves to npm.ps1 which the system execution policy may block; prefer npm.cmd.
$npmCmd = if (Get-Command npm.cmd -ErrorAction SilentlyContinue) { 'npm.cmd' } else { 'npm' }

# 1. Isolated state dir (never touches the Render beta 8088 volume)
$devData = Join-Path $RepoRoot $DataDir
if (-not (Test-Path $devData)) { New-Item -ItemType Directory -Path $devData | Out-Null }
Write-Host "[DEV] isolated DATA_DIR = $devData (Render beta on 8088 UNTOUCHED)"

# 2. .env seeded from template (secrets filled by hand; never printed)
if (-not (Test-Path (Join-Path $RepoRoot ".env"))) {
    if (Test-Path (Join-Path $RepoRoot ".env.example")) {
        Copy-Item (Join-Path $RepoRoot ".env.example") (Join-Path $RepoRoot ".env")
        Write-Host "[DEV] .env created from .env.example — EDIT IT: VAULT_ADDRESS, FAUCET_MNEMONIC, ADMIN_WALLETS"
    } else { Write-Warning ".env.example missing — create .env manually." }
}

# 3. Dev env overlay (structural only; .env holds secrets)
$env:PORT = "$Port"
$env:DATA_DIR = $devData
$env:ALLOWED_ORIGINS = "http://localhost:$Port,http://127.0.0.1:$Port"
$env:DEFAULT_NETWORK = "VOI"
$env:ARENA_STRESS_TEST = "false"

# 4. Build (unless -Prebuilt)
function Build-Wasm {
    Write-Host "[DEV] Building WASM client (main.wasm)..."
    & $npmCmd run wasm:init 2>&1 | Out-Null
    & $npmCmd run wasm:build 2>&1 | Out-Null
    if ($LASTEXITCODE -ne 0) { throw "WASM build failed" }
}
function Build-Sass { & $npmCmd run sass:build 2>&1 | Out-Null; if ($LASTEXITCODE -ne 0) { throw "SASS build failed" } }
function Build-Server {
    Write-Host "[DEV] Building server binary..."
    & $npmCmd run server:build 2>&1 | Out-Null
    if ($LASTEXITCODE -ne 0) { throw "Server build failed" }
}
if (-not $Prebuilt) {
    if ($FullBuild -or -not (Test-Path (Join-Path $RepoRoot "Public\main.wasm"))) { Build-Wasm }
    Build-Sass
    Build-Server
} else { Write-Host "[DEV] -Prebuilt: using existing server binary" }

# Resolve binary name (Go appends .exe on Windows)
$ServerBin = if (Test-Path (Join-Path $RepoRoot "server-bin.exe")) { "server-bin.exe" }
             elseif (Test-Path (Join-Path $RepoRoot "server-bin")) { "server-bin" }
             else { throw "No server binary found. Run without -Prebuilt first." }

# 5. Process tracking (Skill 3: Multi-Service Manager)
$script:ServerProc = $null
$script:BrowserProc = $null

function Write-Log($msg) { Write-Host "[$(Get-Date -Format 'HH:mm:ss')] $msg" }

function Get-ProcessOnPort($port) {
    $line = netstat -ano | Select-String ":$port\s+.*LISTENING"
    if ($line -and $line.Line -match '\s+(\d+)\s*$') { return $Matches[1] }
    return $null
}
function Kill-ProcessOnPort($port) {
    $holderPid = Get-ProcessOnPort $port
    if ($holderPid -and ($script:ServerProc -eq $null -or $holderPid -ne $script:ServerProc.Id)) {
        Write-Log "Killing stale process on port $port (PID $holderPid)"
        taskkill /F /PID $holderPid 2>$null | Out-Null
        Start-Sleep -Seconds 1
    }
}
function Test-Health {
    try {
        $wc = New-Object System.Net.WebClient
        $wc.Headers.Add('Accept', '*/*')
        $null = $wc.DownloadString("http://localhost:$Port/api/faucet/status")
        return $true
    } catch { return $false }
}
function Start-Server {
    Kill-ProcessOnPort $Port
    Write-Log "Starting server on port $Port ($ServerBin)..."
    $script:ServerProc = Start-Process -FilePath (Join-Path $RepoRoot $ServerBin) -PassThru -NoNewWindow
    for ($i = 0; $i -lt 90; $i++) {
        Start-Sleep -Seconds 1
        if (Test-Health) { Write-Log "Server ONLINE (PID $($script:ServerProc.Id)) at http://localhost:$Port"; return $true }
        if ($script:ServerProc.HasExited) { Write-Log "Server exited early (check stderr / dev_server.log)"; return $false }
    }
    Write-Log "Server failed to become healthy in 90s"; return $false
}
function Stop-Server {
    if ($script:ServerProc -and !$script:ServerProc.HasExited) {
        Write-Log "Stopping server (PID $($script:ServerProc.Id))"
        $script:ServerProc.Kill()
        $script:ServerProc.WaitForExit(3000)
    }
    Kill-ProcessOnPort $Port
}
function Find-Chrome {
    $cands = @(
        "${env:ProgramFiles}\Google\Chrome\Application\chrome.exe",
        "${env:ProgramFiles(x86)}\Google\Chrome\Application\chrome.exe",
        "${env:LOCALAPPDATA}\Google\Chrome\Application\chrome.exe",
        "${env:ProgramFiles(x86)}\Microsoft\Edge\Application\msedge.exe",
        "${env:ProgramFiles}\Microsoft\Edge\Application\msedge.exe"
    )
    foreach ($c in $cands) { if (Test-Path $c) { return $c } }
    return $null
}


# 6. Auto-open browser + DevTools (Skill 2: Browser DevTools Bridge)
if (-not $NoOpen) {
    $chrome = Find-Chrome
    if ($chrome) {
        $script:BrowserProc = Start-Process -FilePath $chrome -PassThru -ArgumentList "--auto-open-devtools-for-tabs", "--remote-debugging-port=$CdpPort", "--remote-allow-origins=*", "http://localhost:$Port/"
        Write-Log "Browser opened (DevTools auto-open, CDP on :$CdpPort). Use -NoOpen to disable."
    } else { Write-Warning "No Chrome/Edge found — open http://localhost:$Port/ manually." }
}

# 7. Start server
if (-not (Start-Server)) { Stop-Server; Pop-Location; exit 1 }

# 8. Optional CDP UI/function test harness
if ($Test) {
    Write-Log "Running UI/function test harness against http://localhost:$Port/ ..."
    $env:DEV_PORT = "$CdpPort"
    & node (Join-Path $RepoRoot "tools\server\ui_test_harness.js") --url "http://localhost:$Port/"
    $testRc = $LASTEXITCODE
    if ($testRc -ne 0) { Write-Log "HARNESS reported FAILURES (rc=$testRc)" } else { Write-Log "HARNESS PASSED" }
}

# 9. Watcher + health watchdog (Skills 1 & 4)
function Get-FileSig($path) { if (-not (Test-Path $path)) { return "" }; return (Get-FileHash -Algorithm MD5 $path).Hash }
function Scan-Files {
    $f = @()
    # Public\js\*.js (flat) AND Public\*.js — the second glob is load-bearing: app.js, the
    # composition root, and collective-intelligence.js live at Public/ root, so without it a
    # change to the file that composes the whole app produced NO change event at all.
    $f += Get-ChildItem (Join-Path $RepoRoot "Public\js\*.js") -ErrorAction SilentlyContinue
    $f += Get-ChildItem (Join-Path $RepoRoot "Public\*.js") -ErrorAction SilentlyContinue
    $f += Get-ChildItem (Join-Path $RepoRoot "Public\src\scss\*.scss") -Recurse -ErrorAction SilentlyContinue
    $f += Get-ChildItem (Join-Path $RepoRoot "*.go") -ErrorAction SilentlyContinue
    # index.html is the single entry shell (app-entry-mandate §1); a change to it was silent.
    $f += Get-ChildItem (Join-Path $RepoRoot "Public\*.html") -ErrorAction SilentlyContinue
    $f
}
function Get-Changed {
    $changed = @(); $cur = @{}
    foreach ($file in (Scan-Files)) {
        $h = Get-FileSig $file.FullName; $cur[$file.FullName] = $h
        if ($script:FileHashes.ContainsKey($file.FullName) -and $script:FileHashes[$file.FullName] -ne $h) { $changed += $file }
    }
    $script:FileHashes = $cur; $changed
}
$script:FileHashes = @{}
foreach ($f in (Scan-Files)) { $script:FileHashes[$f.FullName] = Get-FileSig $f.FullName }

if ($NoWatch) {
    Write-Log "NoWatch: server running. Press Ctrl+C to stop."
    try { while ($true) { Start-Sleep -Seconds 5; if (-not $script:ServerProc.HasExited -and -not (Test-Health)) { Write-Log "Server unhealthy"; break } } }
    finally { Stop-Server; if ($script:BrowserProc) { $script:BrowserProc.Kill() }; Pop-Location }
    exit 0
}

Write-Log "Watching for changes (Ctrl+C to stop)..."
try {
    while ($true) {
        Start-Sleep -Seconds 2
        if ($script:ServerProc.HasExited -or -not (Test-Health)) {
            Write-Log "Server unhealthy/down — restarting..."
            Stop-Server
            if (-not (Start-Server)) { Write-Log "Restart failed; will retry next loop" }
            continue
        }
        $changed = Get-Changed
        if ($changed.Count -gt 0) {
            $names = ($changed | ForEach-Object { $_.Name }) -join ", "
            Write-Log "Changed: $names"
            $restart = $false; $sass = $false
            foreach ($f in $changed) {
                switch -Wildcard ($f.Extension) {
                    ".go"   { $restart = $true }
                    ".scss" { $sass = $true }
                    ".js"   { Write-Log "JS changed: $($f.Name) — refresh the browser to see changes" }
                }
            }
            if ($restart) { Stop-Server; Build-Server; Start-Server | Out-Null }
            elseif ($sass) { Build-Sass; Write-Log "SASS rebuilt — refresh the browser" }
        }
    }
} finally {
    Write-Log "Shutting down..."
    Stop-Server
    if ($script:BrowserProc -and !$script:BrowserProc.HasExited) { $script:BrowserProc.Kill() }
    Pop-Location
}

