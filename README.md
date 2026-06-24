# TaskFlow

English | [简体中文](./README_ZH.md)

TaskFlow is a full-stack internal task workflow system. This repository contains:

- a Go HTTP API with explicit task state transitions
- a React frontend under `web/`
- Mongo-backed release, validation, and intranet rollout scripts

The current workflow is centered on a small-team delivery loop:

```text
create -> assign -> start -> submit -> approve/reject -> close
```

## What is in scope

- Password-based login, refresh-token rotation, password reset, account disable, and session revoke
- JWT auth for non-dev environments
- Health, readiness, liveness, metrics, structured logs, request IDs, and optional OTLP tracing
- Mongo migrations, soft delete with audit retention, rate limiting, and idempotency
- A browser UI for login, task list, task detail, task creation, and current-user profile

## Repository layout

```text
taskflow/
├── cmd/                 # server and migration entrypoints
├── internal/
│   ├── bootstrap/       # app assembly
│   ├── config/          # environment config and strict production validation
│   ├── domain/          # aggregates, state machine, ports
│   ├── handler/         # HTTP handlers
│   ├── middleware/      # auth, logging, tracing, rate limit, idempotency
│   ├── repository/      # Mongo and memory adapters
│   ├── router/          # route wiring
│   └── service/         # task and identity use cases
├── web/                 # React + Vite frontend
├── scripts/             # smoke tests, rollout helpers, audits
├── deploy/              # local observability config
├── reports/             # release, security, and performance notes
└── docs/                # project notes and local planning space
```

## Quick start

### 1. Backend only, fastest local loop

Requirements:

- Go `1.25.11`

Run the API in dev mode with the in-memory repository:

```bash
go test ./...

DEV_MODE=true \
TASK_REPOSITORY_DRIVER=memory \
JWT_SECRET=change-me-change-me-change-me-123 \
go run ./cmd/server
```

Dev mode seeds these users:

- `u_test_001` / `creator-pass-123`
- `u_test_002` / `assignee-pass-123`
- `u_agent_001` / `agent-pass-123`

`POST /users` is available in dev mode unless `ALLOW_PUBLIC_REGISTER` is overridden.

### 2. Frontend local preview

Requirements:

- Node `22`

Start the API first, then run the Vite app:

```bash
cd web
npm ci
VITE_API_PROXY_TARGET=http://localhost:8080 npm run dev
```

Open `http://127.0.0.1:5173`.

Useful frontend commands:

```bash
cd web
npm run lint
npm run test
npm run build
npm run preview
```

The frontend uses `/api` by default and the Vite dev server proxies that path to `VITE_API_PROXY_TARGET` or `http://localhost:8080`.

### 3. Local Mongo compose baseline

Bring up the backend stack with Mongo and bootstrap data:

```bash
docker compose up -d --build
bash scripts/compose_smoke.sh
```

If you also want the packaged same-origin web entry:

```bash
docker compose --profile full up -d --build web
bash scripts/nginx_smoke.sh
```

## API surface

Public routes:

- `POST /auth/login`
- `POST /auth/refresh`
- `POST /auth/password-reset/request`
- `POST /auth/password-reset/confirm`
- `POST /users` when public registration is enabled

Authenticated routes:

- `GET /me`
- `GET /users`
- `POST /users/:id/disable`
- `POST /users/:id/revoke-sessions`
- `POST /tasks`
- `GET /tasks`
- `GET /tasks/:id`
- `PATCH /tasks/:id`
- `DELETE /tasks/:id`
- `POST /tasks/:id/{assign,start,submit,reject,approve,close,cancel,reactivate}`
- `GET /tasks/:id/records`
- `GET /tasks/:id/audit_logs`

System routes:

- `GET /health`
- `GET /livez`
- `GET /readyz`
- `GET /metrics`

## Testing and release checks

Backend:

```bash
go test ./...
```

Frontend:

```bash
cd web
npm ci
npm run lint
npm run test
npm run build
```

Repo-level smoke and release helpers:

- `bash scripts/compose_smoke.sh`
- `bash scripts/web_build_smoke.sh`
- `bash scripts/web_acceptance_smoke.sh`
- `bash scripts/nginx_smoke.sh`
- `bash scripts/monitoring_smoke.sh`
- `bash scripts/intranet_acceptance.sh`
- `bash scripts/security_audit.sh`

GitHub Actions runs both frontend and backend checks in [`.github/workflows/ci.yml`](./.github/workflows/ci.yml).

## Deployment and operations

- Deployment baseline: [`DEPLOYMENT.md`](./DEPLOYMENT.md)
- Mongo migration discipline: [`MIGRATIONS.md`](./MIGRATIONS.md)
- Intranet release checklist: [`INTRANET_RELEASE_CHECKLIST.md`](./INTRANET_RELEASE_CHECKLIST.md)
- First deployment runbook: [`INTRANET_RUNBOOK.md`](./INTRANET_RUNBOOK.md)
- Day-2 operations: [`INTRANET_OPS.md`](./INTRANET_OPS.md)
- Acceptance and smoke scripts: [`ACCEPTANCE_TESTING.md`](./ACCEPTANCE_TESTING.md)

## Notes on production behavior

- Production deployments should use `DEV_MODE=false`, `STRICT_PRODUCTION_CONFIG=true`, and `TASK_REPOSITORY_DRIVER=mongo`.
- Mongo transaction-backed flows require a replica set member or `mongos`, not a standalone Mongo server.
- The packaged frontend and API can be served together through the compose `full` profile.
