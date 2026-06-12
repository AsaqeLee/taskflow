#!/usr/bin/env bash
set -euo pipefail

container_name="${TASKFLOW_CONTAINER_NAME:-taskflow}"
previous_image="${TASKFLOW_PREVIOUS_IMAGE:-}"
env_file="${TASKFLOW_ENV_FILE:-.env}"
host_port="${TASKFLOW_PORT:-8080}"
health_url="${TASKFLOW_HEALTH_URL:-http://127.0.0.1:${host_port}/readyz}"
max_attempts="${TASKFLOW_READY_RETRIES:-20}"
sleep_seconds="${TASKFLOW_READY_SLEEP_SECONDS:-3}"

if [[ -z "${previous_image}" ]]; then
  echo "TASKFLOW_PREVIOUS_IMAGE is required" >&2
  exit 1
fi

if [[ ! -f "${env_file}" ]]; then
  echo "TASKFLOW_ENV_FILE does not exist: ${env_file}" >&2
  exit 1
fi

echo "[rollback] stopping current container if it exists"
docker rm -f "${container_name}" >/dev/null 2>&1 || true

echo "[rollback] starting ${previous_image}"
docker run -d \
  --name "${container_name}" \
  --env-file "${env_file}" \
  -p "${host_port}:8080" \
  "${previous_image}" >/dev/null

echo "[rollback] waiting for readiness: ${health_url}"
for ((attempt = 1; attempt <= max_attempts; attempt++)); do
  if curl --silent --fail "${health_url}" >/dev/null; then
    echo "[rollback] service is ready on attempt ${attempt}"
    exit 0
  fi
  sleep "${sleep_seconds}"
done

echo "[rollback] readiness failed, recent container logs:" >&2
docker logs "${container_name}" --tail 100 >&2 || true
exit 1
