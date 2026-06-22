#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
COMPOSE_ARGS=()

refresh_compose_args() {
  local env_file="${COMPOSE_ENV_FILE:-${ENV_FILE:-}}"
  COMPOSE_ARGS=()
  if [[ -n "$env_file" && -f "$env_file" ]]; then
    COMPOSE_ARGS=(--env-file "$env_file")
  fi
}

load_compose_context() {
  local env_file="${COMPOSE_ENV_FILE:-${ENV_FILE:-}}"
  if [[ -n "$env_file" && -f "$env_file" ]]; then
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

compose_down() {
  refresh_compose_args
  if [[ ${#COMPOSE_ARGS[@]} -eq 0 ]]; then
    docker compose down -v --remove-orphans "$@"
  else
    docker compose "${COMPOSE_ARGS[@]}" down -v --remove-orphans "$@"
  fi
}