#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

ENV_FILE="${1:-$ROOT/.env}"
PREVIEW_LOG="${PREVIEW_LOG:-/tmp/taskflow-preview.log}"

log() {
  printf '[pilot-smoke] %s\n' "$*"
}

fail() {
  printf '[pilot-smoke] FAIL: %s\n' "$*" >&2
  exit 1
}

cleanup() {
  if [[ -n "${preview_pid:-}" ]]; then
    kill "$preview_pid" >/dev/null 2>&1 || true
    wait "$preview_pid" >/dev/null 2>&1 || true
  fi
}

trap cleanup EXIT

[[ -f "$ENV_FILE" ]] || fail "env file not found: $ENV_FILE"

export COMPOSE_ENV_FILE="$ENV_FILE"
# shellcheck disable=SC1091
source scripts/compose_env.sh
load_compose_context
WEB_PORT="${WEB_PORT:-8081}"
BASE_URL="${BASE_URL:-http://127.0.0.1:${WEB_PORT}}"

log "validating production env"
bash scripts/validate_production_env.sh "$ENV_FILE"

if [[ ! -f "$ROOT/web/dist/index.html" ]]; then
  log "building frontend bundle"
  bash scripts/web_build_smoke.sh
fi

refresh_compose_args
compose_cmd up -d --build
wait_compose_log bootstrap "bootstrap completed" 40 2 || fail "bootstrap did not report completion"

for attempt in {1..40}; do
  if curl -sf "http://127.0.0.1:8080/readyz" >/dev/null 2>&1; then
    break
  fi
  sleep 3
done

curl -sf "http://127.0.0.1:8080/readyz" >/dev/null || fail "taskflow readyz not healthy"

if [[ ! -d "$ROOT/web/node_modules" ]]; then
  npm --prefix "$ROOT/web" ci
fi

log "serving production bundle via Vite preview on ${BASE_URL}"
rm -f "$PREVIEW_LOG"
npm --prefix "$ROOT/web" run preview -- --host 127.0.0.1 --port "$WEB_PORT" --strictPort >"$PREVIEW_LOG" 2>&1 &
preview_pid="$!"

for attempt in {1..30}; do
  if curl -sf "${BASE_URL}/" >/dev/null 2>&1; then
    break
  fi
  sleep 1
done

curl -sf "${BASE_URL}/" >/dev/null || {
  cat "$PREVIEW_LOG" >&2 || true
  fail "preview server not ready at ${BASE_URL}"
}

login_status=""
for attempt in $(seq 1 5); do
  login_status="$(curl -sS -o /tmp/taskflow_pilot_login.json -w '%{http_code}' \
    -X POST "${BASE_URL}/api/auth/login" \
    -H 'Content-Type: application/json' \
    -d "{\"id\":\"${OWNER_ID}\",\"password\":\"${OWNER_PASS}\"}")"
  if [[ "$login_status" == "200" ]]; then
    break
  fi
  sleep 2
done
[[ "$login_status" == "200" ]] || fail "login via preview /api proxy returned ${login_status}"

log "PASS: production bundle + proxied API login succeeded at ${BASE_URL}"
log "for dockerized nginx entry use: docker compose --profile full up -d (requires nginx image)"
