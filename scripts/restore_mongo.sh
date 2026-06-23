#!/usr/bin/env bash
set -euo pipefail

if [[ $# -lt 1 ]]; then
  echo "usage: $0 <archive.gz> [--drop]" >&2
  echo "  MONGODB_URI, COMPOSE_MONGODB_URI, MONGODB_DATABASE, RESTORE_TOOL override defaults" >&2
  exit 1
fi

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
ARCHIVE="$1"
DROP_FLAG="${2:-}"
MONGODB_URI="${MONGODB_URI:-mongodb://localhost:27018/?replicaSet=rs0}"
COMPOSE_MONGODB_URI="${COMPOSE_MONGODB_URI:-mongodb://127.0.0.1:27017/?replicaSet=rs0}"
MONGODB_DATABASE="${MONGODB_DATABASE:-taskflow}"
RESTORE_TOOL="${RESTORE_TOOL:-auto}"
CHECKSUM_FILE="${ARCHIVE}.sha256"

# shellcheck disable=SC1091
source "$ROOT/scripts/compose_env.sh"
load_compose_context

fail() {
  printf '[restore] FAIL: %s\n' "$*" >&2
  exit 1
}

have_cmd() {
  command -v "$1" >/dev/null 2>&1
}

verify_checksum() {
  local checksum_file="$1"
  if [[ ! -f "$checksum_file" ]]; then
    return 0
  fi

  if have_cmd sha256sum; then
    sha256sum -c "$checksum_file" >/dev/null
  elif have_cmd shasum; then
    shasum -a 256 -c "$checksum_file" >/dev/null
  else
    fail "sha256sum or shasum is required to verify $checksum_file"
  fi
}

restore_with_host_tool() {
  have_cmd mongorestore || fail "mongorestore not found; install MongoDB Database Tools or set RESTORE_TOOL=compose"
  local args=(
    --uri="$MONGODB_URI"
    --nsInclude "${MONGODB_DATABASE}.*"
    --archive="$ARCHIVE"
    --gzip
  )
  if [[ "$DROP_FLAG" == "--drop" ]]; then
    args+=(--drop)
  fi
  mongorestore "${args[@]}"
}

restore_with_compose() {
  local args=(
    exec -T mongo
    mongorestore
    --uri="$COMPOSE_MONGODB_URI"
    --nsInclude "${MONGODB_DATABASE}.*"
    --archive
    --gzip
  )
  if [[ "$DROP_FLAG" == "--drop" ]]; then
    args+=(--drop)
  fi
  compose_cmd "${args[@]}" <"$ARCHIVE"
}

[[ -f "$ARCHIVE" ]] || fail "archive not found: $ARCHIVE"
[[ -z "$DROP_FLAG" || "$DROP_FLAG" == "--drop" ]] || fail "unsupported flag: $DROP_FLAG"

gzip -t "$ARCHIVE"
verify_checksum "$CHECKSUM_FILE"

case "$RESTORE_TOOL" in
  auto)
    if have_cmd mongorestore; then
      restore_with_host_tool
    else
      restore_with_compose
    fi
    ;;
  host)
    restore_with_host_tool
    ;;
  compose)
    restore_with_compose
    ;;
  *)
    fail "unsupported RESTORE_TOOL: $RESTORE_TOOL"
    ;;
esac

printf '[restore] restore complete from %s\n' "$ARCHIVE"
