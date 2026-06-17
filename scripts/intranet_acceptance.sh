#!/usr/bin/env bash
# P3-10 ~ P3-13 intranet MVP acceptance (API-level; browser flow is manual).
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

BASE_URL="${BASE_URL:-http://127.0.0.1:8080}"
MONGO_URI="${MONGODB_URI:-mongodb://127.0.0.1:27018/?replicaSet=rs0}"
MONGO_DB="${MONGODB_DATABASE:-taskflow}"
OWNER_ID="${OWNER_ID:-u_owner}"
OWNER_PASS="${OWNER_PASS:-change-me-owner-123}"
ASSIGNEE_ID="${ASSIGNEE_ID:-u_alice}"
ASSIGNEE_PASS="${ASSIGNEE_PASS:-change-me-alice-123}"
COLD_START="${COLD_START:-0}"
BACKUP_DIR="${BACKUP_DIR:-$ROOT/backups/acceptance}"

log() { printf '[acceptance] %s\n' "$*"; }
fail() { printf '[acceptance] FAIL: %s\n' "$*" >&2; exit 1; }

json_field() {
  python3 - "$1" "$2" <<'PY'
import json, sys
doc = json.load(open(sys.argv[1], encoding="utf-8"))
path = sys.argv[2].split(".")
cur = doc
for part in path:
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
  local user_id="$1" password="$2" out="$3"
  local status
  status="$(curl -sS -o "$out" -w '%{http_code}' \
    -X POST "${BASE_URL}/auth/login" \
    -H 'Content-Type: application/json' \
    -d "{\"id\":\"${user_id}\",\"password\":\"${password}\"}")"
  [[ "$status" == "200" ]] || fail "login ${user_id} returned ${status}"
}

if [[ "$COLD_START" == "1" ]]; then
  log "P3-10 cold start: docker compose down -v"
  docker compose down -v
fi

log "starting stack (migrate + bootstrap + taskflow)"
docker compose up -d --build
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
ids = {u["id"] for u in users}
if sys.argv[2] not in ids:
    raise SystemExit(f"assignee {sys.argv[2]} not in users list")
PY

log "P3-13: owner + assignee full workflow (incl. reject → restart)"
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
  -d "{\"assignee_id\":\"${ASSIGNEE_ID}\"}" >/dev/null

curl -sf -X POST "${BASE_URL}/tasks/${TASK_ID}/start" \
  -H "Authorization: Bearer ${ALICE_TOKEN}" >/dev/null

curl -sf -X POST "${BASE_URL}/tasks/${TASK_ID}/submit" \
  -H "Authorization: Bearer ${ALICE_TOKEN}" \
  -H 'Content-Type: application/json' \
  -d '{"content":"first submit"}' >/dev/null

curl -sf -X POST "${BASE_URL}/tasks/${TASK_ID}/reject" \
  -H "Authorization: Bearer ${OWNER_TOKEN}" \
  -H 'Content-Type: application/json' \
  -d '{"content":"please revise"}' >/dev/null

curl -sf -X POST "${BASE_URL}/tasks/${TASK_ID}/start" \
  -H "Authorization: Bearer ${ALICE_TOKEN}" >/dev/null

curl -sf -X POST "${BASE_URL}/tasks/${TASK_ID}/submit" \
  -H "Authorization: Bearer ${ALICE_TOKEN}" \
  -H 'Content-Type: application/json' \
  -d '{"content":"revised submit"}' >/dev/null

curl -sf -X POST "${BASE_URL}/tasks/${TASK_ID}/approve" \
  -H "Authorization: Bearer ${OWNER_TOKEN}" \
  -H 'Content-Type: application/json' \
  -d '{"content":"LGTM"}' >/dev/null

curl -sf -X POST "${BASE_URL}/tasks/${TASK_ID}/close" \
  -H "Authorization: Bearer ${OWNER_TOKEN}" >/dev/null

FINAL_STATUS="$(curl -sS "${BASE_URL}/tasks/${TASK_ID}" -H "Authorization: Bearer ${OWNER_TOKEN}" | python3 -c 'import json,sys; print(json.load(sys.stdin)["task"]["status"])')"
[[ "$FINAL_STATUS" == "completed" ]] || fail "expected completed, got ${FINAL_STATUS}"

log "P3-11: restart taskflow preserves data"
docker compose restart taskflow
wait_ready
AFTER_RESTART="$(curl -sS "${BASE_URL}/tasks/${TASK_ID}" -H "Authorization: Bearer ${OWNER_TOKEN}" | python3 -c 'import json,sys; print(json.load(sys.stdin)["task"]["status"])')"
[[ "$AFTER_RESTART" == "completed" ]] || fail "task missing after restart"

log "P3-12: backup and restore"
mkdir -p "$BACKUP_DIR"
ARCHIVE="${BACKUP_DIR}/acceptance-$(date +%Y%m%d-%H%M%S).gz"
docker compose exec -T mongo mongodump \
  --uri="mongodb://127.0.0.1:27017/?replicaSet=rs0" \
  --db="$MONGO_DB" \
  --archive \
  --gzip >"$ARCHIVE"
[[ -s "$ARCHIVE" ]] || fail "backup archive empty"

# Drop only the acceptance task to prove restore brings it back.
docker compose exec -T mongo mongosh --quiet "$MONGO_DB" --eval "db.tasks.deleteOne({_id: '${TASK_ID}'})" >/dev/null
GONE_STATUS="$(curl -sS -o /tmp/tf_task_gone.json -w '%{http_code}' \
  "${BASE_URL}/tasks/${TASK_ID}" -H "Authorization: Bearer ${OWNER_TOKEN}" || true)"
[[ "$GONE_STATUS" == "404" ]] || fail "expected 404 after delete, got ${GONE_STATUS}"

docker compose exec -T mongo mongorestore \
  --uri="mongodb://127.0.0.1:27017/?replicaSet=rs0" \
  --archive \
  --gzip \
  --nsInclude="${MONGO_DB}.*" \
  --drop <"$ARCHIVE"

RESTORED_STATUS="$(curl -sS "${BASE_URL}/tasks/${TASK_ID}" -H "Authorization: Bearer ${OWNER_TOKEN}" | python3 -c 'import json,sys; print(json.load(sys.stdin)["task"]["status"])')"
[[ "$RESTORED_STATUS" == "completed" ]] || fail "restore did not bring task back"

log "ALL PASSED: P3-10 (cold=${COLD_START}), P3-11, P3-12, P3-13"