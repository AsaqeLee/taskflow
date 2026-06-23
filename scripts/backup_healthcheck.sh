#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
BACKUP_DIR="${BACKUP_DIR:-$ROOT/backups}"
BACKUP_MAX_AGE_HOURS="${BACKUP_MAX_AGE_HOURS:-26}"

log() {
  printf '[backup-health] %s\n' "$*"
}

fail() {
  printf '[backup-health] FAIL: %s\n' "$*" >&2
  exit 1
}

have_cmd() {
  command -v "$1" >/dev/null 2>&1
}

file_mtime() {
  local file="$1"
  if stat -f %m "$file" >/dev/null 2>&1; then
    stat -f %m "$file"
  else
    stat -c %Y "$file"
  fi
}

verify_checksum() {
  local checksum_file="$1"
  if have_cmd sha256sum; then
    sha256sum -c "$checksum_file" >/dev/null
  elif have_cmd shasum; then
    shasum -a 256 -c "$checksum_file" >/dev/null
  else
    fail "sha256sum or shasum is required"
  fi
}

[[ "$BACKUP_MAX_AGE_HOURS" =~ ^[0-9]+$ ]] || fail "BACKUP_MAX_AGE_HOURS must be a non-negative integer"

latest_archive="$(find "$BACKUP_DIR" -maxdepth 1 -type f -name 'taskflow-*.gz' -print | sort -r | head -n 1)"
[[ -n "$latest_archive" ]] || fail "no backup archives found in $BACKUP_DIR"

checksum_file="${latest_archive}.sha256"
metadata_file="${latest_archive}.metadata"

[[ -f "$checksum_file" ]] || fail "missing checksum file: $checksum_file"
[[ -f "$metadata_file" ]] || fail "missing metadata file: $metadata_file"

gzip -t "$latest_archive"
verify_checksum "$checksum_file"

now_epoch="$(date +%s)"
archive_epoch="$(file_mtime "$latest_archive")"
max_age_seconds=$((BACKUP_MAX_AGE_HOURS * 3600))
age_seconds=$((now_epoch - archive_epoch))

if (( age_seconds > max_age_seconds )); then
  fail "latest backup is older than ${BACKUP_MAX_AGE_HOURS}h: $latest_archive"
fi

log "PASS: latest backup is fresh and valid: $latest_archive"
