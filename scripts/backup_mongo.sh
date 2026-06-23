#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
BACKUP_DIR="${BACKUP_DIR:-$ROOT/backups}"
MONGODB_URI="${MONGODB_URI:-mongodb://localhost:27018/?replicaSet=rs0}"
COMPOSE_MONGODB_URI="${COMPOSE_MONGODB_URI:-mongodb://127.0.0.1:27017/?replicaSet=rs0}"
MONGODB_DATABASE="${MONGODB_DATABASE:-taskflow}"
BACKUP_RETENTION="${BACKUP_RETENTION:-7}"
BACKUP_TOOL="${BACKUP_TOOL:-auto}"
TIMESTAMP="$(date +%Y%m%d-%H%M%S)"
ARCHIVE="$BACKUP_DIR/taskflow-${TIMESTAMP}.gz"
CHECKSUM_FILE="${ARCHIVE}.sha256"
METADATA_FILE="${ARCHIVE}.metadata"

# shellcheck disable=SC1091
source "$ROOT/scripts/compose_env.sh"
load_compose_context

log() {
  printf '[backup] %s\n' "$*"
}

fail() {
  printf '[backup] FAIL: %s\n' "$*" >&2
  exit 1
}

have_cmd() {
  command -v "$1" >/dev/null 2>&1
}

write_checksum() {
  local archive="$1" output="$2"
  if have_cmd sha256sum; then
    sha256sum "$archive" >"$output"
  elif have_cmd shasum; then
    shasum -a 256 "$archive" >"$output"
  else
    fail "sha256sum or shasum is required"
  fi
}

redact_uri() {
  printf '%s' "$1" | sed -E 's#//([^/@]+)@#//***:***@#'
}

backup_with_host_tool() {
  have_cmd mongodump || fail "mongodump not found; install MongoDB Database Tools or set BACKUP_TOOL=compose"
  mongodump \
    --uri="$MONGODB_URI" \
    --db="$MONGODB_DATABASE" \
    --archive="$ARCHIVE" \
    --gzip
}

backup_with_compose() {
  compose_cmd exec -T mongo \
    mongodump \
    --uri="$COMPOSE_MONGODB_URI" \
    --db="$MONGODB_DATABASE" \
    --archive \
    --gzip >"$ARCHIVE"
}

prune_old_archives() {
  [[ "$BACKUP_RETENTION" =~ ^[0-9]+$ ]] || fail "BACKUP_RETENTION must be a non-negative integer"

  archives=()
  while IFS= read -r archive; do
    archives+=("$archive")
  done < <(find "$BACKUP_DIR" -maxdepth 1 -type f -name 'taskflow-*.gz' -print | sort -r)
  if (( ${#archives[@]} <= BACKUP_RETENTION )); then
    return 0
  fi

  for archive in "${archives[@]:BACKUP_RETENTION}"; do
    rm -f "$archive" "${archive}.sha256" "${archive}.metadata"
  done
}

mkdir -p "$BACKUP_DIR"

case "$BACKUP_TOOL" in
  auto)
    if have_cmd mongodump; then
      log "creating backup with host mongodump"
      backup_with_host_tool
      backup_method="host"
    else
      log "creating backup via docker compose exec"
      backup_with_compose
      backup_method="compose"
    fi
    ;;
  host)
    log "creating backup with host mongodump"
    backup_with_host_tool
    backup_method="host"
    ;;
  compose)
    log "creating backup via docker compose exec"
    backup_with_compose
    backup_method="compose"
    ;;
  *)
    fail "unsupported BACKUP_TOOL: $BACKUP_TOOL"
    ;;
esac

gzip -t "$ARCHIVE"
write_checksum "$ARCHIVE" "$CHECKSUM_FILE"

cat >"$METADATA_FILE" <<EOF
timestamp=$TIMESTAMP
archive=$ARCHIVE
database=$MONGODB_DATABASE
tool=$backup_method
mongodb_uri=$(redact_uri "$MONGODB_URI")
compose_mongodb_uri=$(redact_uri "$COMPOSE_MONGODB_URI")
EOF

prune_old_archives

log "backup written: $ARCHIVE"
log "checksum written: $CHECKSUM_FILE"
log "metadata written: $METADATA_FILE"
