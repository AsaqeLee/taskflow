#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

ENV_FILE="${1:-}"

# shellcheck disable=SC1091
source "$ROOT/scripts/compose_env.sh"

log() {
  printf '[release-gate] %s\n' "$*"
}

if [[ -n "$ENV_FILE" ]]; then
  log "validating production env: $ENV_FILE"
  bash scripts/validate_production_env.sh "$ENV_FILE"
  export COMPOSE_ENV_FILE="$ENV_FILE"
  export ENV_FILE
  load_compose_context
else
  refresh_compose_args
fi

log "resetting compose state"
compose_down >/dev/null 2>&1 || true

log "running Go test suite"
go test ./...

log "running frontend lint/test/build smoke"
bash scripts/web_build_smoke.sh

log "running compose smoke"
bash scripts/compose_smoke.sh

log "running browser and responsive acceptance"
bash scripts/web_acceptance_smoke.sh

if [[ -n "$ENV_FILE" ]]; then
  log "running pilot nginx entry smoke"
  bash scripts/pilot_smoke.sh "$ENV_FILE"
fi

compose_down

log "running cold-start intranet acceptance"
COLD_START=1 bash scripts/intranet_acceptance.sh
compose_down

rm -rf web/node_modules web/dist web/test-results

log "checking final diff formatting"
git diff --check

log "PASS: release candidate gate completed"
