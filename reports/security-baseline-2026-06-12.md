# Security Baseline 2026-06-12

## Scope

Capture a reproducible security and dependency baseline aligned with the pinned production toolchain.

## Environment

- Toolchain container: `golang:1.25.11`
- Command entrypoint:

```bash
docker run --rm -v "$PWD":/src -w /src -e RUN_GOSEC=true golang:1.25.11 \
  /bin/sh -lc 'export PATH=/usr/local/go/bin:$PATH; ./scripts/security_audit.sh'
```

## Checks

- `go test ./...`
- `go vet ./...`
- `go run golang.org/x/vuln/cmd/govulncheck@latest ./...`
- `go run github.com/securego/gosec/v2/cmd/gosec@latest ./...`

## Results

- `go test ./...`: pass
- `go vet ./...`: pass
- `govulncheck ./...`: `No vulnerabilities found.`
- `gosec ./...`: `Issues : 0`

## Hardening Adjustments Included In This Baseline

- Runtime and CI toolchain pinned to Go `1.25.11`
- Dev-mode JWT fallback no longer uses a hardcoded static secret
- `_FILE` secret loading now uses a constrained root-based file read helper

## Remaining Gaps

- This is a static baseline, not an external penetration test
- No SBOM export or dependency license report yet
- No secret-manager integration test beyond `_FILE` path loading
