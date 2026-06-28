#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
ENV_FILE="${1:-$ROOT/.env}"

log() {
  printf '[validate-production-env] %s\n' "$*"
}

fail() {
  printf '[validate-production-env] FAIL: %s\n' "$*" >&2
  exit 1
}

trim() {
  local value="$1"
  value="${value#"${value%%[![:space:]]*}"}"
  value="${value%"${value##*[![:space:]]}"}"
  printf '%s' "$value"
}

resolve_path() {
  local raw="$1"
  if [[ "$raw" = /* ]]; then
    printf '%s\n' "$raw"
    return 0
  fi
  printf '%s/%s\n' "$ROOT" "${raw#./}"
}

[[ -f "$ENV_FILE" ]] || fail "env file not found: $ENV_FILE"

set -a
# shellcheck disable=SC1090
source "$ENV_FILE"
set +a

[[ "${DEV_MODE:-false}" == "false" ]] || fail "DEV_MODE must be false"
[[ "${STRICT_PRODUCTION_CONFIG:-false}" == "true" ]] || fail "STRICT_PRODUCTION_CONFIG must be true"
[[ "${TASK_REPOSITORY_DRIVER:-mongo}" == "mongo" ]] || fail "TASK_REPOSITORY_DRIVER must be mongo"
[[ "${ALLOW_PUBLIC_REGISTER:-false}" == "false" ]] || fail "ALLOW_PUBLIC_REGISTER must be false"

jwt_secret_value=""
if [[ -n "${JWT_SECRET_FILE:-}" ]]; then
  jwt_secret_file="$(resolve_path "${JWT_SECRET_FILE}")"
  [[ -f "$jwt_secret_file" ]] || fail "JWT_SECRET_FILE not found: $jwt_secret_file"
  jwt_secret_value="$(tr -d '\r\n' <"$jwt_secret_file")"
else
  jwt_secret_value="${JWT_SECRET:-}"
fi

[[ -n "$jwt_secret_value" ]] || fail "JWT_SECRET or JWT_SECRET_FILE is required"
[[ ${#jwt_secret_value} -ge 32 ]] || fail "JWT secret must be at least 32 characters"
case "$jwt_secret_value" in
  compose-local-secret-must-be-at-least-32-chars|replace-with-openssl-rand-hex-32)
    fail "JWT secret is still using a placeholder value"
    ;;
esac

app_version="$(trim "${APP_VERSION:-}")"
[[ -n "$app_version" ]] || fail "APP_VERSION is required"
case "$app_version" in
  dev|compose-local|replace-with-git-tag-or-short-sha)
    fail "APP_VERSION must be a real release tag or commit SHA"
    ;;
esac

password_reset_webhook_url="$(trim "${PASSWORD_RESET_WEBHOOK_URL:-}")"
[[ -n "$password_reset_webhook_url" ]] || fail "PASSWORD_RESET_WEBHOOK_URL is required"
case "$password_reset_webhook_url" in
  https://*) ;;
  *) fail "PASSWORD_RESET_WEBHOOK_URL must be an absolute https URL" ;;
esac
password_reset_webhook_auth_token="$(trim "${PASSWORD_RESET_WEBHOOK_AUTH_TOKEN:-}")"
if [[ -n "${PASSWORD_RESET_WEBHOOK_AUTH_TOKEN_FILE:-}" ]]; then
  password_reset_webhook_auth_token_file="$(resolve_path "${PASSWORD_RESET_WEBHOOK_AUTH_TOKEN_FILE}")"
  [[ -f "$password_reset_webhook_auth_token_file" ]] || fail "PASSWORD_RESET_WEBHOOK_AUTH_TOKEN_FILE not found: $password_reset_webhook_auth_token_file"
  password_reset_webhook_auth_token="$(tr -d '\r\n' <"$password_reset_webhook_auth_token_file")"
fi
password_reset_webhook_auth_token="$(trim "$password_reset_webhook_auth_token")"
[[ -n "$password_reset_webhook_auth_token" ]] || fail "PASSWORD_RESET_WEBHOOK_AUTH_TOKEN or PASSWORD_RESET_WEBHOOK_AUTH_TOKEN_FILE is required"

bootstrap_file="$(trim "${BOOTSTRAP_USERS_FILE:-}")"
[[ -n "$bootstrap_file" ]] || fail "BOOTSTRAP_USERS_FILE is required"
case "$bootstrap_file" in
  ./scripts/users.example.json|scripts/users.example.json)
    fail "BOOTSTRAP_USERS_FILE must not point at scripts/users.example.json"
    ;;
esac

bootstrap_path="$(resolve_path "$bootstrap_file")"
[[ -f "$bootstrap_path" ]] || fail "bootstrap users file not found: $bootstrap_path"
if grep -n "change-me-" "$bootstrap_path" >/dev/null 2>&1; then
  fail "bootstrap users file still contains change-me-* passwords"
fi

cors_allowed_origins="$(trim "${CORS_ALLOWED_ORIGINS:-}")"
if [[ -n "$cors_allowed_origins" ]]; then
  IFS=',' read -r -a origins <<<"$cors_allowed_origins"
  for raw_origin in "${origins[@]}"; do
    origin="$(trim "$raw_origin")"
    case "$origin" in
      http://localhost*|https://localhost*|http://127.0.0.1*|https://127.0.0.1*|http://[::1]*|https://[::1]*)
        fail "CORS_ALLOWED_ORIGINS still contains local development origin: $origin"
        ;;
    esac
  done
fi

docker compose --env-file "$ENV_FILE" config >/dev/null

log "PASS: production env is internally consistent"
