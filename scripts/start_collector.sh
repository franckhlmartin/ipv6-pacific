#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."
if [[ ! -f go.mod ]] || [[ ! -d cmd/collector ]]; then
  echo "Run from project root (go.mod and cmd/collector required)."
  exit 1
fi
# Examples:
#   ./scripts/start_collector.sh -run-once
#   ./scripts/start_collector.sh -run-once -country=FJ
#   ./scripts/start_collector.sh -country=FJ    # daemon: FJ first, then interval round-robin
echo "Starting collector $*"
exec go run ./cmd/collector/ "$@"
