#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
COMPOSE_ARGS=()

resolve_compose_env_file() {
  local explicit_env="${COMPOSE_ENV_FILE:-${ENV_FILE:-}}"
  if [[ -n "$explicit_env" && -f "$explicit_env" ]]; then
    printf '%s\n' "$explicit_env"
    return 0
  fi

  if [[ -f "$ROOT/.env" ]]; then
    printf '%s\n' "$ROOT/.env"
  fi
}

refresh_compose_args() {
  local env_file
  env_file="$(resolve_compose_env_file)"
  COMPOSE_ARGS=()
  if [[ -n "$env_file" ]]; then
    COMPOSE_ARGS=(--env-file "$env_file")
  fi
}

load_compose_context() {
  local env_file
  env_file="$(resolve_compose_env_file)"
  if [[ -n "$env_file" ]]; then
    set -a
    # shellcheck disable=SC1090
    source "$env_file"
    set +a
    if [[ -n "${BOOTSTRAP_USERS_FILE:-}" ]]; then
      # shellcheck disable=SC1090
      eval "$(bash "$ROOT/scripts/read_bootstrap_credentials.sh" "$BOOTSTRAP_USERS_FILE")"
    fi
  fi
  refresh_compose_args
}

compose_cmd() {
  refresh_compose_args
  if (( ${#COMPOSE_ARGS[@]} == 0 )); then
    docker compose "$@"
  else
    docker compose "${COMPOSE_ARGS[@]}" "$@"
  fi
}

compose_down() {
  compose_cmd down -v --remove-orphans "$@"
}

wait_compose_log() {
  local service="$1" pattern="$2" attempts="${3:-40}" sleep_seconds="${4:-2}"
  local attempt logs
  for attempt in $(seq 1 "$attempts"); do
    logs="$(compose_cmd logs --no-color "$service" 2>/dev/null || true)"
    if grep -q "$pattern" <<<"$logs"; then
      return 0
    fi
    sleep "$sleep_seconds"
  done
  return 1
}
