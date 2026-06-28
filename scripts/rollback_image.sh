#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
ROLLBACK_MODE="${TASKFLOW_ROLLBACK_MODE:-compose}"
TASKFLOW_PREVIOUS_IMAGE="${TASKFLOW_PREVIOUS_IMAGE:-}"
TASKFLOW_ENV_FILE="${TASKFLOW_ENV_FILE:-$ROOT/.env}"
TASKFLOW_ROLLBACK_INCLUDE_WEB="${TASKFLOW_ROLLBACK_INCLUDE_WEB:-false}"
TASKFLOW_CONTAINER_NAME="${TASKFLOW_CONTAINER_NAME:-taskflow}"
TASKFLOW_PORT="${TASKFLOW_PORT:-8080}"
TASKFLOW_HEALTH_URL="${TASKFLOW_HEALTH_URL:-}"
TASKFLOW_READY_RETRIES="${TASKFLOW_READY_RETRIES:-20}"
TASKFLOW_READY_SLEEP_SECONDS="${TASKFLOW_READY_SLEEP_SECONDS:-3}"

# shellcheck disable=SC1091
source "$ROOT/scripts/compose_env.sh"

log() {
  printf '[rollback] %s\n' "$*"
}

fail() {
  printf '[rollback] FAIL: %s\n' "$*" >&2
  exit 1
}

is_true() {
  local normalized
  normalized="$(printf '%s' "${1:-}" | tr '[:upper:]' '[:lower:]')"
  case "$normalized" in
    1|true|yes|on) return 0 ;;
    0|false|no|off|"") return 1 ;;
    *) fail "invalid boolean value: $1" ;;
  esac
}

resolve_path() {
  local raw="$1"
  if [[ "$raw" = /* ]]; then
    printf '%s\n' "$raw"
    return 0
  fi

  printf '%s/%s\n' "$ROOT" "${raw#./}"
}

wait_ready() {
  local ready_url="$1"
  local attempt

  log "waiting readiness: ${ready_url}"
  for attempt in $(seq 1 "$TASKFLOW_READY_RETRIES"); do
    if curl --silent --fail "$ready_url" >/dev/null 2>&1; then
      log "service is ready on attempt ${attempt}"
      return 0
    fi
    sleep "$TASKFLOW_READY_SLEEP_SECONDS"
  done

  return 1
}

rollback_compose() {
  export ENV_FILE="$TASKFLOW_ENV_FILE"
  export COMPOSE_ENV_FILE="$TASKFLOW_ENV_FILE"
  load_compose_context

  export TASKFLOW_IMAGE="$TASKFLOW_PREVIOUS_IMAGE"
  local ready_url="${TASKFLOW_HEALTH_URL:-http://127.0.0.1:${PORT:-$TASKFLOW_PORT}/readyz}"

  if is_true "$TASKFLOW_ROLLBACK_INCLUDE_WEB"; then
    log "restoring compose services with web profile from ${TASKFLOW_PREVIOUS_IMAGE}"
    compose_cmd --profile full up -d --no-build bootstrap taskflow web
  else
    log "restoring compose services from ${TASKFLOW_PREVIOUS_IMAGE}"
    compose_cmd up -d --no-build bootstrap taskflow
  fi

  if wait_ready "$ready_url"; then
    return 0
  fi

  log "readiness failed, recent compose logs:"
  compose_cmd logs --no-color --tail 100 taskflow >&2 || true
  if is_true "$TASKFLOW_ROLLBACK_INCLUDE_WEB"; then
    compose_cmd logs --no-color --tail 100 web >&2 || true
  fi
  exit 1
}

rollback_container() {
  local ready_url="${TASKFLOW_HEALTH_URL:-http://127.0.0.1:${TASKFLOW_PORT}/readyz}"

  log "stopping current container if it exists"
  docker rm -f "$TASKFLOW_CONTAINER_NAME" >/dev/null 2>&1 || true

  log "starting ${TASKFLOW_PREVIOUS_IMAGE}"
  docker run -d \
    --name "$TASKFLOW_CONTAINER_NAME" \
    --env-file "$TASKFLOW_ENV_FILE" \
    -p "${TASKFLOW_PORT}:8080" \
    "$TASKFLOW_PREVIOUS_IMAGE" >/dev/null

  if wait_ready "$ready_url"; then
    return 0
  fi

  log "readiness failed, recent container logs:"
  docker logs "$TASKFLOW_CONTAINER_NAME" --tail 100 >&2 || true
  exit 1
}

TASKFLOW_ENV_FILE="$(resolve_path "$TASKFLOW_ENV_FILE")"

[[ -n "$TASKFLOW_PREVIOUS_IMAGE" ]] || fail "TASKFLOW_PREVIOUS_IMAGE is required"
[[ -f "$TASKFLOW_ENV_FILE" ]] || fail "TASKFLOW_ENV_FILE does not exist: $TASKFLOW_ENV_FILE"

case "$ROLLBACK_MODE" in
  compose)
    rollback_compose
    ;;
  container)
    rollback_container
    ;;
  *)
    fail "unsupported TASKFLOW_ROLLBACK_MODE: $ROLLBACK_MODE"
    ;;
esac
