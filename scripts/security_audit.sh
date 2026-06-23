#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

PROJECT_GO_VERSION="$(awk '/^go / { print $2; exit }' go.mod)"
GO_RUNNER=(env "GOTOOLCHAIN=go${PROJECT_GO_VERSION}" go)

SKIP_GO_TEST="${SKIP_GO_TEST:-false}"
SKIP_GO_VET="${SKIP_GO_VET:-false}"
SKIP_GOVULNCHECK="${SKIP_GOVULNCHECK:-false}"
SKIP_WEB_AUDIT="${SKIP_WEB_AUDIT:-false}"
NPM_AUDIT_LEVEL="${NPM_AUDIT_LEVEL:-high}"
NPM_AUDIT_REGISTRY="${NPM_AUDIT_REGISTRY:-https://registry.npmjs.org}"

echo "[security] go mod verify"
"${GO_RUNNER[@]}" mod verify

if [[ "$SKIP_GO_TEST" != "true" ]]; then
  echo "[security] go test ./..."
  "${GO_RUNNER[@]}" test ./...
fi

if [[ "$SKIP_GO_VET" != "true" ]]; then
  echo "[security] go vet ./..."
  "${GO_RUNNER[@]}" vet ./...
fi

if [[ "$SKIP_GOVULNCHECK" != "true" ]]; then
  echo "[security] govulncheck ./..."
  "${GO_RUNNER[@]}" run golang.org/x/vuln/cmd/govulncheck@latest ./...
fi

if [[ "${RUN_GOSEC:-false}" == "true" ]]; then
  echo "[security] gosec ./..."
  "${GO_RUNNER[@]}" run github.com/securego/gosec/v2/cmd/gosec@latest ./...
fi

if [[ "$SKIP_WEB_AUDIT" != "true" ]]; then
  echo "[security] npm audit web production dependencies"
  npm_config_registry="$NPM_AUDIT_REGISTRY" \
    npm --prefix web audit --audit-level="$NPM_AUDIT_LEVEL" --omit=dev
fi

echo "[security] audit complete"
