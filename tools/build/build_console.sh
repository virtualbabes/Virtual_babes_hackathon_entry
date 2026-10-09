#!/usr/bin/env bash
# build_console.sh — Native console authority build for NFT-Seduction.
# §32.1: console runs the full authoritative sim locally (loopback-only), no cloud model.
set -euo pipefail

TARGET="${1:-windows}"   # windows | linux
OUT="${2:-nft-seduction-console}"

case "$TARGET" in
  windows) GOOS=windows GOARCH=amd64 ;;
  linux)   GOOS=linux   GOARCH=amd64 ;;
  *) echo "usage: $0 [windows|linux] [outname]"; exit 2 ;;
esac

export CGO_ENABLED=0
echo ">> building console target: GOOS=$GOOS GOARCH=$GOARCH -tags console"
GOOS="$GOOS" GOARCH="$GOARCH" go build -tags console -ldflags="-w -s" -o "$OUT" .
echo ">> done: $OUT"
