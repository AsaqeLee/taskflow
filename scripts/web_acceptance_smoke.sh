#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
WEB_DIR="$ROOT/web"

# shellcheck disable=SC1091
source "$ROOT/scripts/compose_env.sh"
load_compose_context
WEB_BASE_URL="${WEB_BASE_URL:-http://127.0.0.1:5173}"
WEB_HOST="${WEB_HOST:-127.0.0.1}"
# Always use the Vite dev port here; do not inherit WEB_PORT from compose pilot env.
WEB_DEV_PORT="${WEB_DEV_PORT:-5173}"
VITE_LOG="${VITE_LOG:-/tmp/taskflow-vite.log}"

log() {
  printf '[web-acceptance] %s\n' "$*"
}

cleanup() {
  if [[ -n "${vite_pid:-}" ]]; then
    kill "$vite_pid" >/dev/null 2>&1 || true
    wait "$vite_pid" >/dev/null 2>&1 || true
  fi
}

trap cleanup EXIT

if [[ ! -d "$WEB_DIR/node_modules" ]]; then
  log "installing frontend dependencies"
  npm --prefix "$WEB_DIR" ci
fi

log "ensuring Playwright chromium is installed"
(
  cd "$WEB_DIR"
  npx playwright install chromium
)

log "starting Vite dev server on ${WEB_BASE_URL}"
rm -f "$VITE_LOG"
npm --prefix "$WEB_DIR" run dev -- --host "$WEB_HOST" --port "$WEB_DEV_PORT" --strictPort >"$VITE_LOG" 2>&1 &
vite_pid="$!"

for attempt in {1..30}; do
  if curl -sf "$WEB_BASE_URL" >/dev/null 2>&1; then
    break
  fi
  sleep 1
done

if ! curl -sf "$WEB_BASE_URL" >/dev/null 2>&1; then
  log "Vite dev server failed to become ready"
  cat "$VITE_LOG" >&2 || true
  exit 1
fi

log "running browser acceptance"
npm --prefix "$WEB_DIR" run acceptance:browser

log "running responsive acceptance"
npm --prefix "$WEB_DIR" run acceptance:responsive

log "PASS: browser and responsive acceptance succeeded"
