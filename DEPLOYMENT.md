# TaskFlow Deployment

## Scope

This repository already supports a practical production-style baseline for internal rollout:

- JWT auth outside dev mode
- refresh-token rotation, password reset, account disable, and session revoke
- request IDs, trace IDs, structured JSON logs, and optional OTLP tracing
- `/health`, `/livez`, `/readyz`, `/metrics`
- Mongo-backed rate limiting and idempotency
- startup or CLI-driven versioned Mongo migrations
- soft delete with audit retention
- an optional same-origin packaged web entry through Docker Compose

Several write paths use Mongo transactions:

- task create, transition, and soft delete
- refresh-token rotation
- password-reset confirmation
- account disable and session revoke side effects

Because of that, production Mongo must be a replica set member or behind `mongos`. A standalone Mongo server is not enough.

## Recommended deployment modes

### Local compose baseline

Use this when you want the backend stack, Mongo, bootstrap users, and smoke coverage on one machine:

```bash
docker compose up -d --build
bash scripts/compose_smoke.sh
```

If you also want the packaged same-origin web entry:

```bash
docker compose --profile full up -d --build web
bash scripts/nginx_smoke.sh
```

### Intranet or pilot deployment

Use a real `.env` file, real bootstrap users, and strict production validation:

```bash
cp .env.intranet.example .env
bash scripts/validate_production_env.sh .env
docker compose up -d --build
```

For a gitignored pilot bundle with generated secrets and initial passwords:

```bash
bash scripts/init_pilot_env.sh
bash scripts/validate_production_env.sh .env
docker compose up -d --build
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
RATE_LIMIT_REQUESTS=120
RATE_LIMIT_WINDOW=1m
LOGIN_RATE_LIMIT_REQUESTS=10
LOGIN_RATE_LIMIT_WINDOW=5m
PASSWORD_RESET_RATE_LIMIT_REQUESTS=5
PASSWORD_RESET_RATE_LIMIT_WINDOW=15m
IDEMPOTENCY_TTL=10m
TRACING_ENABLED=false
TRACING_ENDPOINT=otel-collector:4318
TRACING_INSECURE=true
TRACING_SERVICE_NAME=taskflow
CORS_ALLOWED_ORIGINS=https://taskflow.internal
```

Optional secret-file inputs:

```env
JWT_SECRET_FILE=/run/secrets/taskflow_jwt_secret
MONGODB_URI_FILE=/run/secrets/taskflow_mongodb_uri
```

`STRICT_PRODUCTION_CONFIG=true` rejects unsafe placeholder values, local-development CORS origins, memory mode, and other obviously non-production inputs.

Validate the env file before rollout:

```bash
bash scripts/validate_production_env.sh .env
```

## Build

Build the application image with an immutable version string:

```bash
docker build \
  --build-arg APP_VERSION="$(git rev-parse --short HEAD)" \
  -t taskflow:latest .
```

The repository pins Go `1.25.11` in `go.mod`, and the frontend CI baseline uses Node `22`.

## Run migrations

Apply versioned Mongo migrations before first traffic:

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

The server also applies pending migrations at startup, but running `taskflow-migrate` explicitly keeps rollout order auditable.

## Run the service

### Direct container run

```bash
docker run --rm -p 8080:8080 \
  -e DEV_MODE=false \
  -e TASK_REPOSITORY_DRIVER=mongo \
  -e MONGODB_URI=mongodb://host.docker.internal:27017/?replicaSet=rs0 \
  -e MONGODB_DATABASE=taskflow \
  -e JWT_SECRET=replace-me-with-a-real-secret \
  -e APP_VERSION=local \
  taskflow:latest
```

### Compose-based run

```bash
docker compose up -d --build
```

For the packaged web entry and `/api` proxy:

```bash
docker compose --profile full up -d --build web
```

Compose defaults such as `compose-local` secrets, `compose-local` app version, and local CORS origins are development-only and must be overridden for intranet or production use.

## Health and observability

Important endpoints:

- `GET /livez`: process liveness
- `GET /readyz`: readiness, including Mongo ping when Mongo mode is enabled
- `GET /metrics`: Prometheus-style text metrics

Current response and telemetry behavior:

- every response echoes `X-Request-ID` and `X-Trace-ID`
- metrics include HTTP, identity, rate-limit, and idempotency counters
- when tracing is enabled, OTLP spans line up with trace IDs used in logs

For local monitoring profile validation:

```bash
docker compose --profile monitoring up -d
bash scripts/monitoring_smoke.sh
```

## Rollout sequence

Recommended order:

1. Build and tag the image.
2. Validate `.env` with `scripts/validate_production_env.sh`.
3. Run `taskflow-migrate`.
4. Start the new version.
5. Wait for `/readyz` to return `200`.
6. Run smoke checks before shifting or widening traffic.
7. Keep the previous image available until core checks pass.

Useful post-start checks:

```bash
curl -sf http://127.0.0.1:8080/readyz
bash scripts/compose_smoke.sh
bash scripts/nginx_smoke.sh
bash scripts/web_acceptance_smoke.sh
```

## Rollback

The current migration set is additive, and task deletion is implemented as soft delete, so rollback is operationally straightforward.

Use the helper script for Docker-based rollback:

```bash
TASKFLOW_PREVIOUS_IMAGE=taskflow:previous \
TASKFLOW_ENV_FILE=.env.production \
TASKFLOW_CONTAINER_NAME=taskflow \
TASKFLOW_PORT=8080 \
./scripts/rollback_image.sh
```

Recommended rollback flow:

1. Stop sending traffic to the new version.
2. Shift traffic back to the previous image or container.
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
- Password-reset token delivery still depends on external delivery wiring; in `DEV_MODE=true`, the token is exposed only for local verification.
- OTLP topology is environment-owned; the repo only provides a local collector example.
- CI covers build, tests, smoke, migration, monitoring, and release-audit checks, but it does not provide full CD orchestration.
