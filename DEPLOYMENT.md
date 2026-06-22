# TaskFlow Deployment

## Scope

This repository now supports a minimal production deployment baseline:

- JWT-based authentication
- refresh-token rotation, password reset, account disable, and session revoke
- request IDs / trace IDs
- optional OTLP distributed tracing
- structured JSON logs
- shared global/auth-scoped rate limiting and idempotency keys when Mongo mode is enabled
- `/health`, `/livez`, `/readyz`, `/metrics`
- versioned Mongo migrations via startup or `cmd/migrate`
- soft delete with audit retention

Task write flows and identity-critical flows use Mongo transactions:

- task create / transition / soft delete
- refresh-token rotation
- password-reset confirmation
- account disable (disable + revoke refresh tokens + clear reset tokens)

Production Mongo must therefore be a replica set member or a `mongos` router, not a standalone server.

## Required Environment

Set these at minimum:

```env
PORT=8080
DEV_MODE=false
TASK_REPOSITORY_DRIVER=mongo
MONGODB_URI=mongodb://mongo:27017/?replicaSet=rs0
MONGODB_DATABASE=taskflow
JWT_SECRET=<strong-random-secret>
APP_VERSION=<release-tag>
BOOTSTRAP_USERS_FILE=./scripts/users.intranet.json
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

Optional secret file inputs:

```env
JWT_SECRET_FILE=/run/secrets/taskflow_jwt_secret
MONGODB_URI_FILE=/run/secrets/taskflow_mongodb_uri
```

If you deploy with the checked-in `docker-compose.yml`, the compose stack now reads the same variables from your shell or `.env` file instead of hard-coding `compose-local` values. That includes `JWT_SECRET`, `APP_VERSION`, `CORS_ALLOWED_ORIGINS`, and `BOOTSTRAP_USERS_FILE`.

## Build

```bash
docker build --build-arg APP_VERSION=$(git rev-parse --short HEAD) -t taskflow:latest .
```

The checked-in `go.mod` and `Dockerfile` pin the supported toolchain baseline to Go `1.25.11`.

## Run Database Migration

Before first traffic, apply versioned migrations:

```bash
docker run --rm \
  -e DEV_MODE=false \
  -e TASK_REPOSITORY_DRIVER=mongo \
  -e MONGODB_URI=mongodb://host.docker.internal:27017/?replicaSet=rs0 \
  -e MONGODB_DATABASE=taskflow \
  -e JWT_SECRET=replace-me \
  taskflow:latest /usr/local/bin/taskflow-migrate
```

The server also applies pending migrations at startup. Running `taskflow-migrate` explicitly keeps rollout order auditable.

## Run Service

```bash
docker run --rm -p 8080:8080 \
  -e DEV_MODE=false \
  -e TASK_REPOSITORY_DRIVER=mongo \
  -e MONGODB_URI=mongodb://host.docker.internal:27017/?replicaSet=rs0 \
  -e MONGODB_DATABASE=taskflow \
  -e JWT_SECRET=replace-me \
  -e APP_VERSION=local \
  taskflow:latest
```

## Local Compose Baseline

```bash
docker compose up -d --build
bash scripts/compose_smoke.sh
```

The compose defaults remain local-only:

- `JWT_SECRET=compose-local-secret-must-be-at-least-32-chars`
- `APP_VERSION=compose-local`
- `BOOTSTRAP_USERS_FILE=./scripts/users.example.json`
- `CORS_ALLOWED_ORIGINS=http://localhost:5173,http://127.0.0.1:5173`

Treat these as development defaults only. For intranet/production rollout, override them through `.env` or exported environment variables before `docker compose up`.

## Readiness and Observability

- `GET /livez`: process liveness
- `GET /readyz`: readiness, including Mongo ping when Mongo mode is enabled
- `GET /metrics`: Prometheus-style text metrics
- Prometheus counters include `http_requests_total`, `http_request_duration_seconds_*`, `taskflow_identity_events_total`, `taskflow_rate_limit_decisions_total`, and `taskflow_idempotency_decisions_total`
- `X-Request-ID` and `X-Trace-ID` are echoed on every response
- When tracing is enabled, the service emits OTLP HTTP spans and keeps trace/span IDs aligned with logs

## Rollout

Recommended sequence:

1. Build image with immutable version tag.
2. Run `taskflow-migrate`.
3. Start new version beside old version.
4. Wait for `/readyz` to return `200`.
5. Shift traffic.
6. Keep old version available until smoke checks pass.

## Rollback

Use the provided helper when rolling back a Docker-based deployment:

```bash
TASKFLOW_PREVIOUS_IMAGE=taskflow:previous \
TASKFLOW_ENV_FILE=.env.production \
TASKFLOW_CONTAINER_NAME=taskflow \
TASKFLOW_PORT=8080 \
./scripts/rollback_image.sh
```

Rollback remains low risk because deletes are soft deletes and the current migration set is additive:

1. Stop sending traffic to the new version.
2. Shift traffic back to the previous image tag or run the rollback script above.
3. Keep the same Mongo database; current migrations only add collections and indexes.
4. Inspect retained audit logs plus request/trace IDs for failed requests.

## CI / Security / Performance Baseline

- GitHub Actions workflow: `.github/workflows/ci.yml`
- Security audit helper: `./scripts/security_audit.sh`
- k6 smoke profile: `./scripts/perf_smoke.js`
- Compose smoke helper: `./scripts/compose_smoke.sh`
- Migration discipline: `./MIGRATIONS.md`
- Local collector example: `./deploy/otel-collector.yaml`
- Security baseline report: `./reports/security-baseline-2026-06-12.md`
- Performance baseline report: `./reports/performance-baseline-2026-06-12.md`

## Intranet MVP Rollout

For small-team internal rollout and release gating, see [`INTRANET_RELEASE_CHECKLIST.md`](./INTRANET_RELEASE_CHECKLIST.md), [`INTRANET_RUNBOOK.md`](./INTRANET_RUNBOOK.md), and [`ACCEPTANCE_TESTING.md`](./ACCEPTANCE_TESTING.md).

## Current Limitations

- Refresh tokens are persisted and rotated, but there is no device/session management UI.
- Password reset token delivery is external to this repo; in `DEV_MODE=true` the reset token is echoed for local verification only.
- OTLP tracing is optional and collector/exporter topology is still left to the deployment environment; the repository only ships a local collector baseline.
- CI covers build/test/vet/vulnerability scan plus Mongo service-container and migrate smoke, but it does not yet provide CD orchestration.
