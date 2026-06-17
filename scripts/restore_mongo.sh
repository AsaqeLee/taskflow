#!/usr/bin/env bash
set -euo pipefail

if [[ $# -lt 1 ]]; then
  echo "usage: $0 <archive.gz> [--drop]" >&2
  echo "  MONGODB_URI and MONGODB_DATABASE env vars override defaults" >&2
  exit 1
fi

ARCHIVE="$1"
DROP_FLAG="${2:-}"
MONGODB_URI="${MONGODB_URI:-mongodb://localhost:27018/?replicaSet=rs0}"
MONGODB_DATABASE="${MONGODB_DATABASE:-taskflow}"

if [[ ! -f "$ARCHIVE" ]]; then
  echo "archive not found: $ARCHIVE" >&2
  exit 1
fi

if ! command -v mongorestore >/dev/null 2>&1; then
  echo "mongorestore not found; install MongoDB Database Tools" >&2
  exit 1
fi

ARGS=(--uri="$MONGODB_URI" --archive="$ARCHIVE" --gzip --nsInclude="${MONGODB_DATABASE}.*")
if [[ "$DROP_FLAG" == "--drop" ]]; then
  ARGS+=(--drop)
fi

echo "restoring $ARCHIVE into $MONGODB_DATABASE"
mongorestore "${ARGS[@]}"
echo "restore completed"