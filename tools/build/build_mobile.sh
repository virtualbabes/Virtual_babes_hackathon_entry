#!/bin/bash
# build_mobile.sh — Android / iOS native build for NFT-Seduction (§32.1).
#
# Android: GOOS=android GOARCH=arm64 -tags mobile
# iOS:     GOOS=ios     GOARCH=arm64 -tags mobile
#
# Mobile targets use the SAME console_server.go entrypoint (loopback-only sim).

set -euo pipefail

TARGET="${1:-android}"
OUT="${2:-nft-seduction-mobile}"

# CGO IS PART OF THE TARGET DECISION, NOT A GLOBAL. iOS cannot be built with cgo disabled — the Go
# toolchain refuses it outright: "ios/arm64 requires external (cgo) linking, but cgo is not enabled"
# (measured 2026-09-20). Android builds fine either way and is kept at 0 for a pure-Go artifact.
# The unconditional `export CGO_ENABLED=0` that used to follow this case was REMOVED: it overwrote the
# per-target value and broke iOS.
case "$TARGET" in
    android) export GOOS=android; export GOARCH=arm64; export CGO_ENABLED=0 ;;
    ios)     export GOOS=ios;     export GOARCH=arm64; export CGO_ENABLED=1 ;;
    *)       echo "usage: $0 [android|ios] [outname]" >&2; exit 2 ;;
esac

echo ">> building mobile target: GOOS=$GOOS GOARCH=$GOARCH CGO_ENABLED=$CGO_ENABLED -tags mobile"
go build -tags "console mobile" -ldflags="-w -s" -o "$OUT" .
echo ">> done: $OUT"
