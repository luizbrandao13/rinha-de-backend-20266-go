#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")" && pwd)"
cd "$ROOT"

make -C data-generator

REFS_JSON="${TMPDIR:-/tmp}/rinha-references.json"
gzip -dc resources/references.json.gz > "$REFS_JSON"

time ./data-generator/generate \
  --reuse-refs \
  --refs-in "$REFS_JSON" \
  --payloads-seed 4141 \
  --payloads 54100 \
  --payloads-out test/test-data.json \
  --fraud-ratio-payloads 0.47 \
  --mcc-cfg resources/mcc_risk.json \
  --randomize-payload-dates

echo "Updated test/test-data.json"
jq -r '.references_checksum_sha256, .stats' test/test-data.json
