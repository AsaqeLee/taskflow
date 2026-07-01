#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

PROJECT_GO_VERSION="$(awk '/^go / { print $2; exit }' go.mod)"
GO_RUNNER=(env "GOTOOLCHAIN=go${PROJECT_GO_VERSION}" go)
ENV_FILE="${1:-}"

# shellcheck disable=SC1091
source "$ROOT/scripts/compose_env.sh"

log() {
  printf '[release-gate] %s\n' "$*"
}

cleanup() {
  if [[ -n "${backup_tmpdir:-}" && -d "${backup_tmpdir:-}" ]]; then
    rm -rf "$backup_tmpdir"
  fi
  if [[ -n "${release_gate_env_file:-}" && -f "${release_gate_env_file:-}" ]]; then
    rm -f "$release_gate_env_file"
  fi
  if [[ -n "${release_gate_users_file:-}" && -f "${release_gate_users_file:-}" ]]; then
    rm -f "$release_gate_users_file"
  fi
}

prepare_release_gate_env() {
  local source_env="$1"
  local bootstrap_source=""
  local release_gate_users_rel=""

  release_gate_env_file="$(mktemp)"
  mkdir -p "$ROOT/tmp"

  set -a
  # shellcheck disable=SC1090
  source "$source_env"
  set +a
  bootstrap_source="${BOOTSTRAP_USERS_FILE:-}"

  awk '!/^(MONGODB_URI|BOOTSTRAP_USERS_FILE)=/' "$source_env" >"$release_gate_env_file"
  if [[ -n "$bootstrap_source" ]]; then
    release_gate_users_file="$(mktemp "$ROOT/tmp/release-gate-users.XXXXXX.json")"
    cp "$bootstrap_source" "$release_gate_users_file"
    release_gate_users_rel="./${release_gate_users_file#"$ROOT/"}"
    printf 'BOOTSTRAP_USERS_FILE=%s\n' "$release_gate_users_rel" >>"$release_gate_env_file"
  fi
  printf 'MONGODB_URI=%s\n' "${RELEASE_GATE_COMPOSE_MONGODB_URI:-mongodb://mongo:27017/?replicaSet=rs0}" >>"$release_gate_env_file"
}

trap cleanup EXIT

if [[ -n "$ENV_FILE" ]]; then
  log "validating production env: $ENV_FILE"
  bash scripts/validate_production_env.sh "$ENV_FILE"

  log "preparing compose-compatible gate env"
  prepare_release_gate_env "$ENV_FILE"
  export COMPOSE_ENV_FILE="$release_gate_env_file"
  export ENV_FILE="$release_gate_env_file"
fi

require_container_stack || {
  log "docker preflight failed"
  exit 1
}
load_compose_context

log "resetting compose state"
compose_down >/dev/null 2>&1 || true

log "running Go test suite"
"${GO_RUNNER[@]}" test ./...

log "running frontend lint/test/build smoke"
bash scripts/web_build_smoke.sh

log "running release governance audit"
SKIP_GO_TEST=true bash scripts/security_audit.sh

log "running compose smoke"
bash scripts/compose_smoke.sh

log "running warm-stack intranet acceptance"
STACK_COMPOSE_UP_MODE=skip bash scripts/intranet_acceptance.sh

log "running backup integrity smoke"
backup_tmpdir="$(mktemp -d)"
BACKUP_TOOL=compose BACKUP_DIR="$backup_tmpdir" bash scripts/backup_mongo.sh
BACKUP_DIR="$backup_tmpdir" bash scripts/backup_healthcheck.sh

log "running same-origin nginx smoke"
env STACK_COMPOSE_UP_MODE=skip bash scripts/nginx_smoke.sh "${ENV_FILE:-}"

log "running browser responsive acceptance"
bash scripts/web_acceptance_smoke.sh

log "running monitoring profile smoke"
bash scripts/monitoring_smoke.sh "${ENV_FILE:-}"

compose_down

log "running cold-start intranet acceptance"
COLD_START=1 bash scripts/intranet_acceptance.sh

compose_down

rm -rf web/node_modules web/dist web/test-results

log "checking final diff formatting"
git diff --check

log "PASS: release candidate gate completed"
