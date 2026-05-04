#!/usr/bin/env bash
set -euo pipefail

# shellcheck disable=SC2034
# Production destination (e.g. franck@www.peachymango.org:/opt/peachymango/)
PROD_DEST="franck@www.bookerpal.com:/opt/pacific-monitor/"

cd "$(dirname "$0")/.."
if [[ ! -f go.mod ]] || [[ ! -d cmd/web ]] || [[ ! -d cmd/collector ]]; then
  echo "Run from project root."
  exit 1
fi

INITIAL_DIR=$(pwd)
echo "Building Linux amd64 binaries..."
GOOS=linux GOARCH=amd64 go build -o pacific-web ./cmd/web/
GOOS=linux GOARCH=amd64 go build -o pacific-collector ./cmd/collector/

TEMP_DIR=$(mktemp -d)
trap 'rm -rf "$TEMP_DIR"' EXIT

cp pacific-web pacific-collector "$TEMP_DIR/"
cp -r cmd internal config "$TEMP_DIR/"
mkdir -p "$TEMP_DIR/web"
cp -r cmd/web/static cmd/web/templates "$TEMP_DIR/cmd/web/"
cp go.mod go.sum "$TEMP_DIR/"
[[ -f .env.example ]] && cp .env.example "$TEMP_DIR/"
cp -r scripts docs NOTICE "$TEMP_DIR/" 2>/dev/null || true

echo "Staging layout:"
( cd "$TEMP_DIR" && find . -maxdepth 2 -type d | head -40 )

echo "Rsync to ${PROD_DEST}"
rsync -avz --delete \
  --include='pacific-web' \
  --include='pacific-collector' \
  --include='cmd/' \
  --include='cmd/**' \
  --include='internal/' \
  --include='internal/**' \
  --include='config/' \
  --include='config/**' \
  --include='go.mod' \
  --include='go.sum' \
  --include='.env.example' \
  --include='scripts/' \
  --include='scripts/**' \
  --include='docs/' \
  --include='docs/**' \
  --include='NOTICE' \
  --exclude='*' \
  "$TEMP_DIR/" "$PROD_DEST"

rm -f pacific-web pacific-collector
echo "Done. On server: configure .env, systemd units for pacific-web + pacific-collector, restart services."
