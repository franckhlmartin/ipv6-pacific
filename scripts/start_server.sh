#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."
if [[ ! -f go.mod ]] || [[ ! -d cmd/web ]]; then
  echo "Run from project root (go.mod and cmd/web required)."
  exit 1
fi
echo "Starting HTTPS web server on ${LISTEN:-:8082} (needs certs/ — run ./scripts/gen_dev_certs.sh)..."
exec go run ./cmd/web/
