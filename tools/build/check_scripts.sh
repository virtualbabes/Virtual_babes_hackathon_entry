#!/bin/bash
cd /z/Crypto_Draught/NFT-Seduction
echo "=== check missing script tags ==="
for f in Public/js/*.js; do
  name=$(basename "$f")
  if ! grep -q "$name" Public/index.html; then
    echo "MISSING: $name"
  fi
done
