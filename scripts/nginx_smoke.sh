#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

ENV_FILE="${1:-}"
WEB_PORT="${WEB_PORT:-8081}"
BASE_URL="${NGINX_BASE_URL:-http://127.0.0.1:${WEB_PORT}}"
HEADERS_FILE="${HEADERS_FILE:-/tmp/taskflow-nginx-headers.txt}"
compose_up_mode="${STACK_COMPOSE_UP_MODE:-build}"

# shellcheck disable=SC1091
source "$ROOT/scripts/compose_env.sh"

log() {
  printf '[nginx-smoke] %s\n' "$*"
}

fail() {
  printf '[nginx-smoke] FAIL: %s\n' "$*" >&2
  exit 1
}

if [[ -n "$ENV_FILE" ]]; then
  export COMPOSE_ENV_FILE="$ENV_FILE"
  export ENV_FILE
fi
load_compose_context

OWNER_ID="${OWNER_ID:-u_owner}"
OWNER_PASS="${OWNER_PASS:-change-me-owner-123}"

case "$compose_up_mode" in
  build)
    if [[ ! -f "$ROOT/web/dist/index.html" ]]; then
      log "building frontend bundle"
      bash scripts/web_build_smoke.sh
    fi
    compose_cmd --profile full up -d --build web
    ;;
  no-build)
    compose_cmd --profile full up -d --no-build web
    ;;
  skip)
    log "skipping compose up; checking existing full-profile stack"
    ;;
  *)
    fail "unsupported STACK_COMPOSE_UP_MODE=$compose_up_mode"
    ;;
esac
wait_compose_log bootstrap "bootstrap completed" 40 2 || fail "bootstrap did not report completion"

for attempt in {1..30}; do
  if curl -sf "${BASE_URL}/" >/dev/null 2>&1; then
    break
  fi
  sleep 2
done

curl -sf "${BASE_URL}/" >/dev/null || fail "nginx web entry not ready at ${BASE_URL}"
curl -sSI "${BASE_URL}/" | tr -d '\r' >"$HEADERS_FILE"

grep -qi '^x-content-type-options: nosniff$' "$HEADERS_FILE" || fail "missing X-Content-Type-Options header"
grep -qi '^x-frame-options: DENY$' "$HEADERS_FILE" || fail "missing X-Frame-Options header"
grep -qi '^referrer-policy: strict-origin-when-cross-origin$' "$HEADERS_FILE" || fail "missing Referrer-Policy header"
grep -qi '^cross-origin-opener-policy: same-origin$' "$HEADERS_FILE" || fail "missing Cross-Origin-Opener-Policy header"
grep -qi '^permissions-policy: camera=(), geolocation=(), microphone=()$' "$HEADERS_FILE" || fail "missing Permissions-Policy header"
grep -qi "^content-security-policy: .*default-src 'self'" "$HEADERS_FILE" || fail "missing Content-Security-Policy header"

login_status=""
for attempt in $(seq 1 5); do
  login_status="$(curl -sS -o /tmp/taskflow_nginx_login.json -w '%{http_code}' \
    -X POST "${BASE_URL}/api/auth/login" \
    -H 'Content-Type: application/json' \
    -d "{\"id\":\"${OWNER_ID}\",\"password\":\"${OWNER_PASS}\"}")"
  if [[ "$login_status" == "200" ]]; then
    break
  fi
  sleep 2
done

[[ "$login_status" == "200" ]] || fail "same-origin login via nginx returned ${login_status}"

log "PASS: nginx same-origin entry and hardened response headers validated at ${BASE_URL}"
