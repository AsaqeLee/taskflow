#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

ENV_FILE="${1:-}"
PROM_QUERY_PATH="${PROM_QUERY_PATH:-/api/v1/query?query=up%7Bjob%3D%22taskflow%22%7D}"

# shellcheck disable=SC1091
source "$ROOT/scripts/compose_env.sh"

log() {
  printf '[monitoring-smoke] %s\n' "$*"
}

fail() {
  printf '[monitoring-smoke] FAIL: %s\n' "$*" >&2
  exit 1
}

prometheus_http() {
  compose_cmd --profile monitoring exec -T prometheus \
    wget -qO- "http://127.0.0.1:9090$1"
}

alertmanager_http() {
  compose_cmd --profile monitoring exec -T alertmanager \
    wget -qO- "http://127.0.0.1:9093$1"
}

json_has_healthy_taskflow_target() {
  python3 -c '
import json
import sys

payload = json.load(sys.stdin)
for item in payload.get("data", {}).get("result", []):
    value = item.get("value", [])
    if len(value) >= 2 and value[1] == "1":
        raise SystemExit(0)
raise SystemExit(1)
'
}

json_has_required_rules() {
  python3 -c '
import json
import sys

payload = json.load(sys.stdin)
groups = payload.get("data", {}).get("groups", [])
rule_names = {
    rule.get("name")
    for group in groups
    for rule in group.get("rules", [])
}
required = {
    "TaskflowDown",
    "TaskflowHigh5xxRate",
    "TaskflowLoginRateLimitSpike",
    "TaskflowProcessRestarted",
}
missing = required - rule_names
if missing:
    raise SystemExit("missing rules: " + ", ".join(sorted(missing)))
'
}

if [[ -n "$ENV_FILE" ]]; then
  export COMPOSE_ENV_FILE="$ENV_FILE"
  export ENV_FILE
fi

load_compose_context
compose_cmd --profile monitoring up -d prometheus alertmanager

for attempt in {1..90}; do
  if prometheus_http "/-/ready" >/dev/null 2>&1 && alertmanager_http "/-/ready" >/dev/null 2>&1; then
    break
  fi
  sleep 2
done

prometheus_http "/-/ready" >/dev/null || fail "prometheus not ready via compose network"
alertmanager_http "/-/ready" >/dev/null || fail "alertmanager not ready via compose network"

for attempt in {1..60}; do
  if prometheus_http "$PROM_QUERY_PATH" | json_has_healthy_taskflow_target; then
    break
  fi
  sleep 2
done

prometheus_http "$PROM_QUERY_PATH" | json_has_healthy_taskflow_target \
  || fail "prometheus did not scrape taskflow metrics"
prometheus_http "/api/v1/rules" | json_has_required_rules \
  || fail "taskflow alert rules not loaded in prometheus"
alertmanager_http "/api/v2/status" >/dev/null || fail "alertmanager status endpoint failed"

log "PASS: Prometheus scraped taskflow metrics and loaded taskflow alert rules"
