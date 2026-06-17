#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
USERS_FILE="${USERS_FILE:-$ROOT/scripts/users.example.json}"

if [[ ! -f "$USERS_FILE" ]]; then
  echo "users file not found: $USERS_FILE" >&2
  exit 1
fi

cd "$ROOT"
export USERS_FILE
go run ./cmd/bootstrap -users "$USERS_FILE"