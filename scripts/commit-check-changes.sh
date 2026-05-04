#!/usr/bin/env bash
# List changes since the last push: fetch origin, then diff current branch vs its upstream
# (or origin/<branch> if no upstream is set).
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

if ! git rev-parse --git-dir >/dev/null 2>&1; then
  echo "Not a git repository." >&2
  exit 1
fi

echo "Fetching from origin..."
git fetch origin

branch="$(git rev-parse --abbrev-ref HEAD)"
if git rev-parse "@{u}" >/dev/null 2>&1; then
  upstream="@{u}"
else
  upstream="origin/${branch}"
  if ! git rev-parse "${upstream}" >/dev/null 2>&1; then
    echo "No upstream branch and no ${upstream}. Set tracking with:" >&2
    echo "  git branch -u origin/${branch}" >&2
    exit 1
  fi
fi

echo ""
echo "Branch: ${branch}"
echo "Comparing to: ${upstream}"
echo "Files changed (name-status) — what would be included in the next push:"
echo "---"
git diff --name-status "${upstream}..HEAD"
