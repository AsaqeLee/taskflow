#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

resolve_path() {
  local raw="$1"
  if [[ "$raw" = /* ]]; then
    printf '%s\n' "$raw"
    return 0
  fi
  printf '%s/%s\n' "$ROOT" "${raw#./}"
}

users_file="${1:-${BOOTSTRAP_USERS_FILE:-./scripts/users.example.json}}"
users_path="$(resolve_path "$users_file")"

[[ -f "$users_path" ]] || {
  printf 'read_bootstrap_credentials: file not found: %s\n' "$users_path" >&2
  exit 1
}

python3 - "$users_path" <<'PY'
import json
import shlex
import sys

users = json.load(open(sys.argv[1], encoding="utf-8"))
by_id = {user["id"]: user.get("password", "") for user in users}

owner = by_id.get("u_owner", "")
assignee = by_id.get("u_alice", "")

if not owner:
    raise SystemExit("bootstrap users file missing u_owner password")

exports = {
    "OWNER_ID": "u_owner",
    "OWNER_PASS": owner,
    "OWNER_PASSWORD": owner,
    "STACK_USER_ID": "u_owner",
    "STACK_USER_PASSWORD": owner,
    "ASSIGNEE_ID": "u_alice",
    "ASSIGNEE_PASS": assignee or owner,
    "ASSIGNEE_PASSWORD": assignee or owner,
}

for key, value in exports.items():
    print(f"export {key}={shlex.quote(value)}")
PY