# build_mobile.ps1 — Android / iOS native build for NFT-Seduction (§32.1).
#
# Android: GOOS=android GOARCH=arm64 -tags mobile
# iOS:     GOOS=ios     GOARCH=arm64 -tags mobile
#
# Mobile targets use the SAME console_server.go entrypoint (loopback-only sim).
# They get their OWN build LATER, on a SIMILAR SCALE (tiered power figure +
# backend auto-cap + identity alias). Until then, mobile clients use a
# SERVER-HOSTED instance (alias served from a gateway the client streams).

param(
    [string]$Target = "android",   # android | ios
    [string]$Out = "nft-seduction-mobile"
)

$ErrorActionPreference = "Stop"

switch ($Target) {
    "android" { $env:GOOS = "android"; $env:GOARCH = "arm64"; $env:CGO_ENABLED = "0"; $ext = "" }
    # iOS REQUIRES cgo — the Go toolchain refuses the target outright with cgo disabled:
    #   "ios/arm64 requires external (cgo) linking, but cgo is not enabled"
    # (measured on this host 2026-09-20). The value set here is therefore LOAD-BEARING and is no
    # longer overwritten below; doing so was the defect. HONEST LIMIT: an ios build still needs an
    # Apple toolchain, so it cannot be exercised from Windows — only the requirement is proven here.
    "ios"     { $env:GOOS = "ios";     $env:GOARCH = "arm64"; $env:CGO_ENABLED = "1"; $ext = "" }
    default   { Write-Error "usage: .\build_mobile.ps1 [android|ios] [outname]"; exit 2 }
}

# (The unconditional `$env:CGO_ENABLED = "0"` that used to sit here was REMOVED 2026-09-20: it
# overwrote the per-target decision above, forcing cgo off for iOS — the one target that cannot build
# without it. Android: measured rc 0, a 32,365,482 B arm64 artifact. The switch is now the sole owner.)
$output = "$Out$ext"
Write-Host ">> building mobile target: GOOS=$env:GOOS GOARCH=$env:GOARCH -tags mobile"
go build -tags "console mobile" -ldflags="-w -s" -o $output .
Write-Host ">> done: $output"
