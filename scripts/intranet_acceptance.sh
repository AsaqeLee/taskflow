#!/usr/bin/env bash
# P2 + P3-10 ~ P3-13 intranet acceptance (API-level; browser flow is separate).
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

# shellcheck disable=SC1091
source "$ROOT/scripts/compose_env.sh"

BASE_URL="${BASE_URL:-http://127.0.0.1:8080}"
MONGO_DB="${MONGODB_DATABASE:-taskflow}"
OWNER_ID="${OWNER_ID:-u_owner}"
OWNER_PASS="${OWNER_PASS:-change-me-owner-123}"
ASSIGNEE_ID="${ASSIGNEE_ID:-u_alice}"
ASSIGNEE_PASS="${ASSIGNEE_PASS:-change-me-alice-123}"
COLD_START="${COLD_START:-0}"
BACKUP_DIR="${BACKUP_DIR:-$ROOT/backups/acceptance}"
STACK_COMPOSE_UP_MODE="${STACK_COMPOSE_UP_MODE:-build}"
HERMES_ID="u_hermes_acceptance"
HERMES_NAME="Hermes Acceptance Agent"
HERMES_PASS="Hermes-Accept-456"

log() {
  printf '[acceptance] %s\n' "$*"
}

fail() {
  printf '[acceptance] FAIL: %s\n' "$*" >&2
  exit 1
}

json_field() {
  python3 - "$1" "$2" <<'PY'
import json, sys

doc = json.load(open(sys.argv[1], encoding="utf-8"))
cur = doc
for part in sys.argv[2].split("."):
    cur = cur[part]
print(cur)
PY
}

wait_ready() {
  local attempt
  for attempt in $(seq 1 40); do
    if curl -sf "${BASE_URL}/readyz" >/dev/null 2>&1; then
      return 0
    fi
    sleep 3
  done
  fail "readyz not healthy at ${BASE_URL}"
}

login() {
  local user_id="$1"
  local password="$2"
  local out="$3"
  local attempt
  local status=""

  for attempt in $(seq 1 5); do
    status="$(curl -sS -o "$out" -w '%{http_code}' \
      -X POST "${BASE_URL}/auth/login" \
      -H 'Content-Type: application/json' \
      -d "{\"id\":\"${user_id}\",\"password\":\"${password}\"}")"
    if [[ "$status" == "200" ]]; then
      return 0
    fi
    sleep 2
  done

  fail "login ${user_id} returned ${status}"
}

start_stack() {
  case "$STACK_COMPOSE_UP_MODE" in
    build)
      compose_cmd up -d --build
      ;;
    no-build)
      compose_cmd up -d --no-build
      ;;
    skip)
      log "skipping compose up; checking existing stack"
      ;;
    *)
      fail "unsupported STACK_COMPOSE_UP_MODE=${STACK_COMPOSE_UP_MODE}"
      ;;
  esac
}

require_container_stack || fail "docker / docker compose unavailable"
load_compose_context

if [[ "$COLD_START" == "1" && "$STACK_COMPOSE_UP_MODE" == "skip" ]]; then
  fail "COLD_START=1 cannot be combined with STACK_COMPOSE_UP_MODE=skip"
fi

if [[ "$COLD_START" == "1" ]]; then
  log "P3-10 cold start: docker compose down -v"
  compose_down
fi

log "starting stack (migrate + bootstrap + taskflow)"
start_stack
wait_compose_log bootstrap "bootstrap completed" 40 2 || fail "bootstrap did not report completion"
wait_ready

log "P3-10: bootstrap users can login"
login "$OWNER_ID" "$OWNER_PASS" /tmp/tf_owner_login.json
login "$ASSIGNEE_ID" "$ASSIGNEE_PASS" /tmp/tf_alice_login.json
OWNER_TOKEN="$(json_field /tmp/tf_owner_login.json access_token)"
ALICE_TOKEN="$(json_field /tmp/tf_alice_login.json access_token)"
curl -sf -H "Authorization: Bearer ${OWNER_TOKEN}" "${BASE_URL}/me" >/dev/null

log "P1: public register is closed"
ANON_REG_STATUS="$(curl -sS -o /tmp/tf_anon_reg.json -w '%{http_code}' \
  -X POST "${BASE_URL}/users" \
  -H 'Content-Type: application/json' \
  -d '{"id":"u_should_fail","name":"Anon","role":"human","password":"strong-pass-123"}')"
[[ "$ANON_REG_STATUS" == "401" ]] || fail "anonymous register expected 401, got ${ANON_REG_STATUS}"

log "P1: owner can list active users"
USERS_STATUS="$(curl -sS -o /tmp/tf_users.json -w '%{http_code}' \
  -H "Authorization: Bearer ${OWNER_TOKEN}" \
  "${BASE_URL}/users?active=true")"
[[ "$USERS_STATUS" == "200" ]] || fail "GET /users expected 200, got ${USERS_STATUS}"
python3 - <<'PY' /tmp/tf_users.json "${ASSIGNEE_ID}"
import json, sys

users = json.load(open(sys.argv[1], encoding="utf-8"))["users"]
ids = {user["id"] for user in users}
if sys.argv[2] not in ids:
    raise SystemExit(f"assignee {sys.argv[2]} not in users list")
PY

log "P2: owner can provision Hermes API key"
HERMES_CREATE_STATUS="$(curl -sS -o /tmp/tf_hermes_create.json -w '%{http_code}' \
  -X POST "${BASE_URL}/users" \
  -H "Authorization: Bearer ${OWNER_TOKEN}" \
  -H 'Content-Type: application/json' \
  -d "{\"id\":\"${HERMES_ID}\",\"name\":\"${HERMES_NAME}\",\"role\":\"agent\",\"password\":\"${HERMES_PASS}\"}")"
if [[ "$HERMES_CREATE_STATUS" != "201" && "$HERMES_CREATE_STATUS" != "409" ]]; then
  fail "create Hermes agent expected 201/409, got ${HERMES_CREATE_STATUS}"
fi

HERMES_KEY_STATUS="$(curl -sS -o /tmp/tf_hermes_api_key.json -w '%{http_code}' \
  -X POST "${BASE_URL}/users/${HERMES_ID}/api-keys" \
  -H "Authorization: Bearer ${OWNER_TOKEN}" \
  -H 'Content-Type: application/json' \
  -d '{"name":"acceptance-hermes"}')"
[[ "$HERMES_KEY_STATUS" == "201" ]] || fail "create Hermes API key expected 201, got ${HERMES_KEY_STATUS}"

HERMES_KEY="$(json_field /tmp/tf_hermes_api_key.json key)"
HERMES_KEY_ID="$(json_field /tmp/tf_hermes_api_key.json api_key.id)"
HERMES_ME_STATUS="$(curl -sS -o /tmp/tf_hermes_me.json -w '%{http_code}' \
  -H "Authorization: Bearer ${HERMES_KEY}" \
  "${BASE_URL}/me")"
[[ "$HERMES_ME_STATUS" == "200" ]] || fail "Hermes API key /me expected 200, got ${HERMES_ME_STATUS}"
python3 - <<'PY' /tmp/tf_hermes_me.json "${HERMES_ID}"
import json, sys

user = json.load(open(sys.argv[1], encoding="utf-8"))["user"]
if user["id"] != sys.argv[2]:
    raise SystemExit(f"expected Hermes id {sys.argv[2]}, got {user['id']}")
PY

log "P3-13: owner + Hermes API key full workflow (incl. reject -> restart)"
TASK_STATUS="$(curl -sS -o /tmp/tf_task_create.json -w '%{http_code}' \
  -X POST "${BASE_URL}/tasks" \
  -H "Authorization: Bearer ${OWNER_TOKEN}" \
  -H 'Content-Type: application/json' \
  -d '{"title":"Acceptance Flow Task","description":"intranet acceptance"}')"
[[ "$TASK_STATUS" == "201" ]] || fail "create task returned ${TASK_STATUS}"

TASK_ID="$(json_field /tmp/tf_task_create.json task.id)"
curl -sf -X POST "${BASE_URL}/tasks/${TASK_ID}/assign" \
  -H "Authorization: Bearer ${OWNER_TOKEN}" \
  -H 'Content-Type: application/json' \
  -d "{\"assignee_id\":\"${HERMES_ID}\"}" >/dev/null
curl -sf -X POST "${BASE_URL}/tasks/${TASK_ID}/start" \
  -H "Authorization: Bearer ${HERMES_KEY}" >/dev/null
curl -sf -X POST "${BASE_URL}/tasks/${TASK_ID}/submit" \
  -H "Authorization: Bearer ${HERMES_KEY}" \
  -H 'Content-Type: application/json' \
  -d '{"content":"first submit"}' >/dev/null
curl -sf -X POST "${BASE_URL}/tasks/${TASK_ID}/reject" \
  -H "Authorization: Bearer ${OWNER_TOKEN}" \
  -H 'Content-Type: application/json' \
  -d '{"content":"please revise"}' >/dev/null
curl -sf -X POST "${BASE_URL}/tasks/${TASK_ID}/start" \
  -H "Authorization: Bearer ${HERMES_KEY}" >/dev/null
curl -sf -X POST "${BASE_URL}/tasks/${TASK_ID}/submit" \
  -H "Authorization: Bearer ${HERMES_KEY}" \
  -H 'Content-Type: application/json' \
  -d '{"content":"revised submit"}' >/dev/null
curl -sf -X POST "${BASE_URL}/tasks/${TASK_ID}/approve" \
  -H "Authorization: Bearer ${OWNER_TOKEN}" \
  -H 'Content-Type: application/json' \
  -d '{"content":"LGTM"}' >/dev/null
curl -sf -X POST "${BASE_URL}/tasks/${TASK_ID}/close" \
  -H "Authorization: Bearer ${OWNER_TOKEN}" >/dev/null

FINAL_STATUS="$(curl -sS "${BASE_URL}/tasks/${TASK_ID}" \
  -H "Authorization: Bearer ${OWNER_TOKEN}" | \
  python3 -c 'import json,sys; print(json.load(sys.stdin)["task"]["status"])')"
[[ "$FINAL_STATUS" == "completed" ]] || fail "expected completed, got ${FINAL_STATUS}"

log "P3-11: restart taskflow preserves data"
compose_cmd restart taskflow
wait_ready
AFTER_RESTART="$(curl -sS "${BASE_URL}/tasks/${TASK_ID}" \
  -H "Authorization: Bearer ${OWNER_TOKEN}" | \
  python3 -c 'import json,sys; print(json.load(sys.stdin)["task"]["status"])')"
[[ "$AFTER_RESTART" == "completed" ]] || fail "task missing after restart"

log "P3-12: backup restore"
mkdir -p "$BACKUP_DIR"
ARCHIVE="${BACKUP_DIR}/acceptance-$(date +%Y%m%d-%H%M%S).gz"
compose_cmd exec -T mongo mongodump \
  --uri="mongodb://127.0.0.1:27017/?replicaSet=rs0" \
  --db="$MONGO_DB" \
  --archive \
  --gzip >"$ARCHIVE"
[[ -s "$ARCHIVE" ]] || fail "backup archive empty"

# Delete only the acceptance task and prove the backup restores it.
compose_cmd exec -T mongo mongosh --quiet "$MONGO_DB" \
  --eval "db.tasks.deleteOne({_id: '${TASK_ID}'})" >/dev/null
GONE_STATUS="$(curl -sS -o /tmp/tf_task_gone.json -w '%{http_code}' \
  "${BASE_URL}/tasks/${TASK_ID}" \
  -H "Authorization: Bearer ${OWNER_TOKEN}" || true)"
[[ "$GONE_STATUS" == "404" ]] || fail "expected 404 after delete, got ${GONE_STATUS}"

compose_cmd exec -T mongo mongorestore \
  --uri="mongodb://127.0.0.1:27017/?replicaSet=rs0" \
  --archive \
  --gzip \
  --nsInclude="${MONGO_DB}.*" \
  --drop <"$ARCHIVE"
RESTORED_STATUS="$(curl -sS "${BASE_URL}/tasks/${TASK_ID}" \
  -H "Authorization: Bearer ${OWNER_TOKEN}" | \
  python3 -c 'import json,sys; print(json.load(sys.stdin)["task"]["status"])')"
[[ "$RESTORED_STATUS" == "completed" ]] || fail "restore did not bring task back"

log "P2: revoke Hermes API key"
HERMES_REVOKE_STATUS="$(curl -sS -o /tmp/tf_hermes_revoke.json -w '%{http_code}' \
  -X POST "${BASE_URL}/users/${HERMES_ID}/api-keys/${HERMES_KEY_ID}/revoke" \
  -H "Authorization: Bearer ${OWNER_TOKEN}")"
[[ "$HERMES_REVOKE_STATUS" == "200" ]] || fail "revoke Hermes API key expected 200, got ${HERMES_REVOKE_STATUS}"

HERMES_REUSE_STATUS="$(curl -sS -o /tmp/tf_hermes_reuse.json -w '%{http_code}' \
  -H "Authorization: Bearer ${HERMES_KEY}" \
  "${BASE_URL}/me" || true)"
[[ "$HERMES_REUSE_STATUS" == "401" ]] || fail "revoked Hermes API key expected 401, got ${HERMES_REUSE_STATUS}"

log "ALL PASSED: P2, P3-10 (cold=${COLD_START}), P3-11, P3-12, P3-13"
