# TaskFlow Deployment

## Scope

This repository now supports a minimal production deployment baseline:

- JWT-based authentication
- refresh-token rotation, password reset, and account disable
- request IDs / trace IDs
- optional OTLP distributed tracing
- structured JSON logs
- shared rate limiting and idempotency keys when Mongo mode is enabled
- `/health`, `/livez`, `/readyz`, `/metrics`
- versioned Mongo migrations via startup or `cmd/migrate`
- soft delete with audit retention

## Required Environment

Set these at minimum:

```env
PORT=8080
DEV_MODE=false
TASK_REPOSITORY_DRIVER=mongo
MONGODB_URI=mongodb://mongo:27017
MONGODB_DATABASE=taskflow
JWT_SECRET=<strong-random-secret>
APP_VERSION=<release-tag>
REFRESH_TOKEN_TTL=168h
PASSWORD_RESET_TTL=1h
TRACING_ENABLED=false
TRACING_ENDPOINT=otel-collector:4318
TRACING_INSECURE=true
TRACING_SERVICE_NAME=taskflow
```

Optional secret file inputs:

```env
JWT_SECRET_FILE=/run/secrets/taskflow_jwt_secret
MONGODB_URI_FILE=/run/secrets/taskflow_mongodb_uri
```

## Build

```bash
docker build --build-arg APP_VERSION=$(git rev-parse --short HEAD) -t taskflow:latest .
```

## Run Database Migration

Before first traffic, apply versioned migrations:

```bash
docker run --rm \
  -e DEV_MODE=false \
  -e TASK_REPOSITORY_DRIVER=mongo \
  -e MONGODB_URI=mongodb://host.docker.internal:27017 \
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
  -e MONGODB_URI=mongodb://host.docker.internal:27017 \
  -e MONGODB_DATABASE=taskflow \
  -e JWT_SECRET=replace-me \
  -e APP_VERSION=local \
  taskflow:latest
```

## Readiness and Observability

- `GET /livez`: process liveness
- `GET /readyz`: readiness, including Mongo ping when Mongo mode is enabled
- `GET /metrics`: Prometheus-style text metrics
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

## Current Limitations

- Refresh tokens are persisted and rotated, but there is no device/session management UI.
- Password reset token delivery is external to this repo; in `DEV_MODE=true` the reset token is echoed for local verification only.
- OTLP tracing is optional and collector/exporter topology is left to the deployment environment.
- CI covers build/test/vet/vulnerability scan, but does not yet run a live Mongo-backed deployment job.
