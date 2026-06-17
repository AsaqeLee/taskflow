#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT/web"

npm ci
npm run lint
npm run test
npm run build

test -f dist/index.html
test -d dist/assets
echo "[web-smoke] lint, test, and production build succeeded"