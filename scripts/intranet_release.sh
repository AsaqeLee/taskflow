#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

ENV_FILE="${1:-$ROOT/.env}"
INCLUDE_WEB="${TASKFLOW_RELEASE_INCLUDE_WEB:-false}"
RUN_WEB_ACCEPTANCE="${TASKFLOW_RELEASE_RUN_WEB_ACCEPTANCE:-false}"
ROLLBACK_ON_FAILURE="${TASKFLOW_RELEASE_ROLLBACK_ON_FAILURE:-true}"
DRY_RUN="${TASKFLOW_RELEASE_DRY_RUN:-false}"
READY_RETRIES="${TASKFLOW_RELEASE_READY_RETRIES:-30}"
READY_SLEEP_SECONDS="${TASKFLOW_RELEASE_READY_SLEEP_SECONDS:-3}"

# shellcheck disable=SC1091
source "$ROOT/scripts/compose_env.sh"

log() {
  printf '[intranet-release] %s\n' "$*"
}

fail() {
  printf '[intranet-release] FAIL: %s\n' "$*" >&2
  exit 1
}

is_true() {
  local normalized
  normalized="$(printf '%s' "${1:-}" | tr '[:upper:]' '[:lower:]')"
  case "$normalized" in
    1|true|yes|on)
      return 0
      ;;
    0|false|no|off|"")
      return 1
      ;;
    *)
      fail "invalid boolean value: $1"
      ;;
  esac
}

run_cmd() {
  if is_true "$DRY_RUN"; then
    printf '[intranet-release] DRY_RUN'
    printf ' %q' "$@"
    printf '\n'
    return 0
  fi
  "$@"
}

wait_ready() {
  local ready_url="$1"
  if is_true "$DRY_RUN"; then
    log "DRY_RUN skip readiness probe: ${ready_url}"
    return 0
  fi

  local attempt
  for attempt in $(seq 1 "$READY_RETRIES"); do
    if curl --silent --fail "$ready_url" >/dev/null 2>&1; then
      log "service ready on attempt ${attempt}"
      return 0
    fi
    sleep "$READY_SLEEP_SECONDS"
  done
  return 1
}

capture_rollback_image() {
  local container_id image_id

  container_id="$(compose_cmd ps -q taskflow 2>/dev/null || true)"
  if [[ -z "$container_id" ]]; then
    log "no existing taskflow container found; rollback image not captured"
    return 0
  fi

  image_id="$(docker inspect -f '{{.Image}}' "$container_id")"
  rollback_image="taskflow:rollback-$(date +%Y%m%d%H%M%S)"
  run_cmd docker tag "$image_id" "$rollback_image"
  log "captured rollback image ${rollback_image}"
}

rollback_release() {
  if is_true "$DRY_RUN"; then
    log "DRY_RUN skip rollback to ${rollback_image:-<none>}"
    return 0
  fi
  if ! is_true "$ROLLBACK_ON_FAILURE"; then
    log "release failed; rollback disabled"
    return 0
  fi
  if [[ -z "${rollback_image:-}" ]]; then
    log "release failed; no rollback image available"
    return 0
  fi

  log "rolling back to ${rollback_image}"
  env \
    TASKFLOW_ROLLBACK_MODE=compose \
    TASKFLOW_PREVIOUS_IMAGE="$rollback_image" \
    TASKFLOW_ENV_FILE="$ENV_FILE" \
    TASKFLOW_HEALTH_URL="$ready_url" \
    TASKFLOW_READY_RETRIES="$READY_RETRIES" \
    TASKFLOW_READY_SLEEP_SECONDS="$READY_SLEEP_SECONDS" \
    TASKFLOW_ROLLBACK_INCLUDE_WEB="$INCLUDE_WEB" \
    bash "$ROOT/scripts/rollback_image.sh" || log "rollback helper failed"
}

on_exit() {
  local status=$?
  if [[ $status -eq 0 ]]; then
    return 0
  fi
  if [[ "${candidate_deployed:-false}" != "true" ]]; then
    return "$status"
  fi
  rollback_release || true
  return "$status"
}

trap on_exit EXIT

[[ -f "$ENV_FILE" ]] || fail "env file not found: $ENV_FILE"

export COMPOSE_ENV_FILE="$ENV_FILE"
export ENV_FILE

log "validating production env"
bash scripts/validate_production_env.sh "$ENV_FILE"

set -a
# shellcheck disable=SC1090
source "$ENV_FILE"
set +a
load_compose_context

if is_true "$RUN_WEB_ACCEPTANCE"; then
  INCLUDE_WEB=true
fi

app_version="${APP_VERSION:-}"
[[ -n "$app_version" ]] || fail "APP_VERSION is required"
safe_app_version="$(printf '%s' "$app_version" | tr -c 'A-Za-z0-9_.-' '-')"
candidate_image="${TASKFLOW_RELEASE_IMAGE:-taskflow:release-${safe_app_version}}"
ready_url="${TASKFLOW_RELEASE_READY_URL:-http://127.0.0.1:${PORT:-8080}/readyz}"
rollback_image=""
candidate_deployed=false

log "candidate image: ${candidate_image}"
capture_rollback_image

if is_true "$INCLUDE_WEB"; then
  log "building packaged frontend bundle"
  run_cmd bash scripts/web_build_smoke.sh
fi

log "building release image"
run_cmd docker build --build-arg "APP_VERSION=${app_version}" -t "$candidate_image" .

export TASKFLOW_IMAGE="$candidate_image"

log "starting base services"
run_cmd compose_cmd up -d mongo mongo-init otel-collector

log "running migration"
run_cmd compose_cmd up --no-build migrate

log "deploying application"
if is_true "$INCLUDE_WEB"; then
  run_cmd compose_cmd --profile full up -d --no-build bootstrap taskflow web
else
  run_cmd compose_cmd up -d --no-build bootstrap taskflow
fi
candidate_deployed=true

log "waiting for readiness"
wait_ready "$ready_url" || fail "readyz did not return success: ${ready_url}"

log "running API smoke"
run_cmd env STACK_COMPOSE_UP_MODE=skip bash scripts/compose_smoke.sh

if is_true "$INCLUDE_WEB"; then
  log "running same-origin nginx smoke"
  run_cmd env STACK_COMPOSE_UP_MODE=skip bash scripts/nginx_smoke.sh "$ENV_FILE"
fi

if is_true "$RUN_WEB_ACCEPTANCE"; then
  log "running browser acceptance smoke"
  run_cmd bash scripts/web_acceptance_smoke.sh
fi

log "release completed successfully"
