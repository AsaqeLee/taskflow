#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
BACKUP_DIR="${BACKUP_DIR:-$ROOT/backups}"
MONGODB_URI="${MONGODB_URI:-mongodb://localhost:27018/?replicaSet=rs0}"
MONGODB_DATABASE="${MONGODB_DATABASE:-taskflow}"
TIMESTAMP="$(date +%Y%m%d-%H%M%S)"
ARCHIVE="$BACKUP_DIR/taskflow-${TIMESTAMP}.gz"

mkdir -p "$BACKUP_DIR"

if ! command -v mongodump >/dev/null 2>&1; then
  echo "mongodump not found; install MongoDB Database Tools" >&2
  exit 1
fi

mongodump \
  --uri="$MONGODB_URI" \
  --db="$MONGODB_DATABASE" \
  --archive="$ARCHIVE" \
  --gzip

echo "backup written: $ARCHIVE"

# Keep the newest 7 archives by default.
if [[ "${BACKUP_RETENTION:-7}" =~ ^[0-9]+$ ]]; then
  ls -1t "$BACKUP_DIR"/taskflow-*.gz 2>/dev/null | tail -n +$((BACKUP_RETENTION + 1)) | xargs rm -f 2>/dev/null || true
fi