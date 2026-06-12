#!/usr/bin/env bash
set -euo pipefail

base_url="${BASE_URL:-http://127.0.0.1:8080}"
user_id="${STACK_USER_ID:-u_stack_demo}"
password="${STACK_USER_PASSWORD:-stack-pass-123}"
max_attempts="${STACK_READY_RETRIES:-30}"
sleep_seconds="${STACK_READY_SLEEP_SECONDS:-3}"

docker compose up -d --build

for ((attempt = 1; attempt <= max_attempts; attempt++)); do
  if curl --silent --fail "${base_url}/readyz" >/dev/null; then
    break
  fi
  sleep "${sleep_seconds}"
done

curl --silent --show-error --fail \
  -X POST "${base_url}/users" \
  -H 'Content-Type: application/json' \
  -d "{\"id\":\"${user_id}\",\"name\":\"Stack Demo\",\"role\":\"human\",\"password\":\"${password}\"}" \
  >/dev/null || true

login_status="$(curl --silent --show-error -o /tmp/taskflow_login_response.json -w '%{http_code}' \
  -X POST "${base_url}/auth/login" \
  -H 'Content-Type: application/json' \
  -d "{\"id\":\"${user_id}\",\"password\":\"${password}\"}")"

if [[ "${login_status}" != "200" ]]; then
  echo "[compose-smoke] login failed with status ${login_status}" >&2
  cat /tmp/taskflow_login_response.json >&2 || true
  exit 1
fi

metrics_status="$(curl --silent --show-error -o /tmp/taskflow_metrics_response.txt -w '%{http_code}' "${base_url}/metrics")"
if [[ "${metrics_status}" != "200" ]]; then
  echo "[compose-smoke] metrics fetch failed with status ${metrics_status}" >&2
  exit 1
fi

echo "[compose-smoke] stack is ready, login succeeded, metrics endpoint responded"
echo "[compose-smoke] inspect traces with: docker compose logs otel-collector --tail 50"
