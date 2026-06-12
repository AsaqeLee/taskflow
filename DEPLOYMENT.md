# TaskFlow Deployment

## Scope

This repository now supports a minimal production deployment baseline:

- JWT-based authentication
- request IDs / trace IDs
- structured JSON logs
- rate limiting
- idempotency keys
- `/health`, `/livez`, `/readyz`, `/metrics`
- Mongo index bootstrap via startup or `cmd/migrate`
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

## Run Index Migration

Before first traffic, ensure indexes:

```bash
docker run --rm \
  -e DEV_MODE=false \
  -e TASK_REPOSITORY_DRIVER=mongo \
  -e MONGODB_URI=mongodb://host.docker.internal:27017 \
  -e MONGODB_DATABASE=taskflow \
  -e JWT_SECRET=replace-me \
  taskflow:latest /usr/local/bin/taskflow-migrate
```

The server also attempts to create required indexes at startup.

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

## Rollout

Recommended sequence:

1. Build image with immutable version tag.
2. Run `taskflow-migrate`.
3. Start new version beside old version.
4. Wait for `/readyz` to return `200`.
5. Shift traffic.
6. Keep old version available until smoke checks pass.

## Rollback

Because deletes are soft deletes and schema changes are index-only in this pass, rollback is simple:

1. Stop sending traffic to the new version.
2. Shift traffic back to the previous image tag.
3. Keep the same Mongo database and indexes; index additions are backward-compatible.
4. Inspect retained audit logs for failed requests using `request_id` / `trace_id`.

## Current Limitations

- Rate limiting and idempotency storage are in-process only; multi-instance deployments need shared storage for strict global guarantees.
- No refresh-token flow yet.
- Mongo collections are index-managed, not versioned by a full migration framework.
