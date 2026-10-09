# build_console.ps1 — Native console authority build for NFT-Seduction (Windows).
# §32.1: console runs the full authoritative sim locally (loopback-only), no cloud model.
param(
    [string]$Target = "windows",   # windows | linux
    [string]$Out = "nft-seduction-console"
)

$ErrorActionPreference = "Stop"

switch ($Target) {
    "windows" { $env:GOOS = "windows"; $env:GOARCH = "amd64"; $ext = ".exe" }
    "linux"   { $env:GOOS = "linux";   $env:GOARCH = "amd64"; $ext = "" }
    default   { Write-Error "usage: .\build_console.ps1 [windows|linux] [outname]"; exit 2 }
}

$env:CGO_ENABLED = "0"
$output = "$Out$ext"
Write-Host ">> building console target: GOOS=$env:GOOS GOARCH=$env:GOARCH -tags console"
go build -tags console -ldflags="-w -s" -o $output .
Write-Host ">> done: $output"
