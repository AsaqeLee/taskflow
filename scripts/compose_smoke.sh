#!/usr/bin/env bash
set -euo pipefail

base_url="${BASE_URL:-http://127.0.0.1:8080}"
max_attempts="${STACK_READY_RETRIES:-30}"
sleep_seconds="${STACK_READY_SLEEP_SECONDS:-3}"

docker compose up -d --build

for ((attempt = 1; attempt <= max_attempts; attempt++)); do
  if curl --silent --fail "${base_url}/readyz" >/dev/null; then
    break
  fi
  sleep "${sleep_seconds}"
done

user_id="${STACK_USER_ID:-u_owner}"
password="${STACK_USER_PASSWORD:-change-me-owner-123}"

login_status="$(curl --silent --show-error -o /tmp/taskflow_login_response.json -w '%{http_code}' \
  -X POST "${base_url}/auth/login" \
  -H 'Content-Type: application/json' \
  -d "{\"id\":\"${user_id}\",\"password\":\"${password}\"}")"

if [[ "${login_status}" != "200" ]]; then
  echo "[compose-smoke] login failed with status ${login_status}" >&2
  cat /tmp/taskflow_login_response.json >&2 || true
  exit 1
fi

access_token="$(python3 - <<'PY'
import json

with open('/tmp/taskflow_login_response.json', 'r', encoding='utf-8') as fp:
    print(json.load(fp)["access_token"])
PY
)"

task_status="$(curl --silent --show-error -o /tmp/taskflow_task_response.json -w '%{http_code}' \
  -X POST "${base_url}/tasks" \
  -H "Authorization: Bearer ${access_token}" \
  -H 'Content-Type: application/json' \
  -d '{"title":"Compose Smoke Task","description":"transactional write path"}')"
if [[ "${task_status}" != "201" ]]; then
  echo "[compose-smoke] task create failed with status ${task_status}" >&2
  cat /tmp/taskflow_task_response.json >&2 || true
  exit 1
fi

task_list_status="$(curl --silent --show-error -o /tmp/taskflow_task_list_response.json -w '%{http_code}' \
  -X GET "${base_url}/tasks" \
  -H "Authorization: Bearer ${access_token}")"
if [[ "${task_list_status}" != "200" ]]; then
  echo "[compose-smoke] task list failed with status ${task_list_status}" >&2
  cat /tmp/taskflow_task_list_response.json >&2 || true
  exit 1
fi

metrics_status="$(curl --silent --show-error -o /tmp/taskflow_metrics_response.txt -w '%{http_code}' "${base_url}/metrics")"
if [[ "${metrics_status}" != "200" ]]; then
  echo "[compose-smoke] metrics fetch failed with status ${metrics_status}" >&2
  exit 1
fi

echo "[compose-smoke] stack is ready, login and task write path succeeded, metrics endpoint responded"
echo "[compose-smoke] inspect traces with: docker compose logs otel-collector --tail 50"
