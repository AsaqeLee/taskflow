# TaskFlow Deployment

## Scope

The repository supports a practical intranet deployment baseline:

- JWT auth outside dev mode
- refresh-token rotation, password reset, account disable, session revoke
- request IDs, trace IDs, structured logs, optional OTLP tracing
- `/health`, `/livez`, `/readyz`, `/metrics`
- Mongo-backed rate limiting and idempotency
- versioned Mongo migrations
- soft delete with audit retention
- optional same-origin packaged web entry through Docker Compose

Several write paths use Mongo transactions, including task transitions, refresh-token rotation, password-reset confirmation, and account/session side effects. Production Mongo must therefore be a replica set member or sit behind `mongos`; standalone Mongo is not sufficient.

## Recommended deployment modes

### Local compose baseline

```bash
docker compose up -d --build
bash scripts/compose_smoke.sh
```

For the packaged same-origin web entry:

```bash
docker compose --profile full up -d --build web
bash scripts/nginx_smoke.sh
```

### Intranet or pilot deployment

Use a real `.env`, real bootstrap users, and strict production validation:

```bash
cp .env.intranet.example .env
bash scripts/validate_production_env.sh .env
```

For a gitignored pilot bundle with generated secrets and initial passwords:

```bash
bash scripts/init_pilot_env.sh
bash scripts/validate_production_env.sh .env
```

## Required environment

Minimum recommended settings:

```env
PORT=8080
WEB_PORT=8081
DEV_MODE=false
ALLOW_PUBLIC_REGISTER=false
STRICT_PRODUCTION_CONFIG=true
TASK_REPOSITORY_DRIVER=mongo
MONGODB_URI=mongodb://mongo:27017/?replicaSet=rs0
MONGODB_DATABASE=taskflow
JWT_SECRET=<strong-random-secret>
APP_VERSION=<release-tag-or-short-sha>
BOOTSTRAP_USERS_FILE=./scripts/users.intranet.json
LOG_LEVEL=info
ACCESS_TOKEN_TTL=2h
REFRESH_TOKEN_TTL=168h
PASSWORD_RESET_TTL=1h
PASSWORD_RESET_WEBHOOK_URL=https://mailer.internal/hooks/taskflow-password-reset
PASSWORD_RESET_WEBHOOK_AUTH_TOKEN_FILE=/run/secrets/taskflow_reset_webhook_token
CORS_ALLOWED_ORIGINS=https://taskflow.internal
```

Optional secret-file inputs:

```env
JWT_SECRET_FILE=/run/secrets/taskflow_jwt_secret
MONGODB_URI_FILE=/run/secrets/taskflow_mongodb_uri
PASSWORD_RESET_WEBHOOK_AUTH_TOKEN_FILE=/run/secrets/taskflow_reset_webhook_token
```

Validate the final env file before any rollout:

```bash
bash scripts/validate_production_env.sh .env
```

## Build and migrations

Build an immutable candidate image:

```bash
docker build \
  --build-arg APP_VERSION="$(git rev-parse --short HEAD)" \
  -t taskflow:latest .
```

Run migrations explicitly when you need an auditable rollout order:

```bash
docker run --rm \
  -e DEV_MODE=false \
  -e TASK_REPOSITORY_DRIVER=mongo \
  -e MONGODB_URI=mongodb://host.docker.internal:27017/?replicaSet=rs0 \
  -e MONGODB_DATABASE=taskflow \
  -e JWT_SECRET=replace-me-with-a-real-secret \
  taskflow:latest \
  /usr/local/bin/taskflow-migrate
```

The server also applies pending migrations on startup, but the dedicated migrate entrypoint is preferred for controlled releases.

## Controlled release path

Compose defaults such as `compose-local` secrets, `compose-local` app version, and local CORS origins are development-only and must be overridden for intranet or production use.

Candidate gate before any real rollout:

```bash
bash scripts/release_candidate_check.sh .env
```

This gate is the mandatory place for:

- Go / frontend / governance checks
- warm-stack API acceptance
- cold-start acceptance
- Hermes API key lifecycle verification
- backup / restore verification
- nginx, browser, and monitoring smoke

Controlled deployment entry:

```bash
bash scripts/intranet_release.sh .env
```

For the packaged same-origin web entry:

```bash
TASKFLOW_RELEASE_INCLUDE_WEB=true bash scripts/intranet_release.sh .env
```

When browser verification must also run on that environment:

```bash
TASKFLOW_RELEASE_INCLUDE_WEB=true \
TASKFLOW_RELEASE_RUN_WEB_ACCEPTANCE=true \
bash scripts/intranet_release.sh .env
```

`scripts/intranet_release.sh` intentionally runs non-destructive post-deploy smoke only. It does not execute the destructive backup/restore branch from `scripts/intranet_acceptance.sh`.

## Health and observability

Important endpoints:

- `GET /livez`: process liveness
- `GET /readyz`: readiness, including Mongo ping when Mongo mode is enabled
- `GET /metrics`: Prometheus-style text metrics

Current response telemetry behavior:

- every response echoes `X-Request-ID` and `X-Trace-ID`
- metrics include HTTP, identity, rate-limit, idempotency counters
- when tracing is enabled, OTLP spans line up with trace IDs used in logs

For local monitoring profile validation:

```bash
docker compose --profile monitoring up -d
bash scripts/monitoring_smoke.sh
```

## Rollout sequence

Recommended order:

1. Validate `.env` with `bash scripts/validate_production_env.sh .env`.
2. Run `bash scripts/release_candidate_check.sh .env` in a candidate environment with Docker / Compose available.
3. Run `bash scripts/intranet_release.sh .env` on the target environment as the only supported deployment entry.
4. If shipping the same-origin web entry, set `TASKFLOW_RELEASE_INCLUDE_WEB=true`; add `TASKFLOW_RELEASE_RUN_WEB_ACCEPTANCE=true` when browser verification is required on that host.
5. Keep the previous image available until post-release checks pass.

Useful post-start checks:

```bash
curl -sf http://127.0.0.1:8080/readyz
bash scripts/compose_smoke.sh
bash scripts/nginx_smoke.sh
bash scripts/web_acceptance_smoke.sh
```

## Rollback

The current migration set is additive, and task deletion is soft delete, so rollback is operationally straightforward.

Use the helper script for compose-based rollback:

```bash
TASKFLOW_PREVIOUS_IMAGE=taskflow:previous \
TASKFLOW_ENV_FILE=.env.production \
TASKFLOW_ROLLBACK_INCLUDE_WEB=true \
./scripts/rollback_image.sh
```

For legacy single-container deployments, set `TASKFLOW_ROLLBACK_MODE=container`.

Recommended rollback flow:

1. Stop sending traffic to the new version.
2. Shift traffic back to the previous compose image.
3. Reuse the same Mongo database.
4. Inspect logs, request IDs, trace IDs, and retained audit history for failed requests.

## Supporting documents

- Migration discipline: [`MIGRATIONS.md`](./MIGRATIONS.md)
- Intranet release checklist: [`INTRANET_RELEASE_CHECKLIST.md`](./INTRANET_RELEASE_CHECKLIST.md)
- First deployment runbook: [`INTRANET_RUNBOOK.md`](./INTRANET_RUNBOOK.md)
- Day-2 operations: [`INTRANET_OPS.md`](./INTRANET_OPS.md)
- Acceptance scripts: [`ACCEPTANCE_TESTING.md`](./ACCEPTANCE_TESTING.md)
- Local collector example: [`deploy/otel-collector.yaml`](./deploy/otel-collector.yaml)
- Security baseline: [`reports/security-baseline-2026-06-12.md`](./reports/security-baseline-2026-06-12.md)
- Performance baseline: [`reports/performance-baseline-2026-06-12.md`](./reports/performance-baseline-2026-06-12.md)

## Current limitations

- There is no device/session management UI yet.
- Password-reset token delivery still depends on external webhook wiring; in `DEV_MODE=true`, the token is exposed only for local verification.
- OTLP topology is environment-owned; the repo only provides a local collector example.
- CI covers build, tests, smoke, migration, monitoring, and release-audit checks, but does not provide full CD orchestration.
