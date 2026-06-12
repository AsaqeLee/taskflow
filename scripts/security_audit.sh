#!/usr/bin/env bash
set -euo pipefail

echo "[security] go test ./..."
go test ./...

echo "[security] go vet ./..."
go vet ./...

echo "[security] govulncheck ./..."
go run golang.org/x/vuln/cmd/govulncheck@latest ./...

if [[ "${RUN_GOSEC:-false}" == "true" ]]; then
  echo "[security] gosec ./..."
  go run github.com/securego/gosec/v2/cmd/gosec@latest ./...
fi

echo "[security] audit complete"
