# Performance Baseline 2026-06-12

## Scope

Validate that the hardened runtime remains stable under light concurrent load after pinning the runtime toolchain to Go `1.25.11`.

## Environment

- App image built from the repository `Dockerfile`
- Go runtime baseline: `1.25.11`
- MongoDB: `mongo:7`
- OTLP collector: `otel/opentelemetry-collector-contrib:0.130.1`
- k6 script: `scripts/perf_smoke.js`

## Runs

### 1. Compose stack baseline

Command:

```bash
docker exec taskflow-mongo-1 mongosh --quiet mongodb://127.0.0.1:27017/taskflow --eval 'db.runtime_rate_limits.deleteMany({}); db.runtime_idempotency_keys.deleteMany({});'
docker run --rm --network taskflow_default -v "$PWD/scripts:/scripts" \
  -e BASE_URL=http://taskflow:8080 \
  -e VUS=4 \
  -e DURATION=14s \
  grafana/k6:latest run /scripts/perf_smoke.js
```

Result:

- `http_req_failed=0.00%`
- `http_req_duration p(95)=13.29ms`
- `http_reqs=113`
- `iterations=56`

Notes:

- This run stays below the default `RATE_LIMIT_REQUESTS=120` budget so it validates the shipped compose defaults without triggering intentional throttling.

### 2. Tracing regression check under prior crash profile

Command:

```bash
docker rm -f taskflow-highlimit >/dev/null 2>&1 || true
docker run -d --name taskflow-highlimit --network taskflow_default \
  -e PORT=8080 \
  -e DEV_MODE=false \
  -e TASK_REPOSITORY_DRIVER=mongo \
  -e MONGODB_URI=mongodb://mongo:27017 \
  -e MONGODB_DATABASE=taskflow \
  -e JWT_SECRET=compose-local-secret \
  -e APP_VERSION=compose-local \
  -e LOG_LEVEL=info \
  -e ACCESS_TOKEN_TTL=2h \
  -e REFRESH_TOKEN_TTL=168h \
  -e PASSWORD_RESET_TTL=1h \
  -e REQUEST_TIMEOUT=15s \
  -e SHUTDOWN_TIMEOUT=10s \
  -e SERVER_READ_TIMEOUT=10s \
  -e SERVER_WRITE_TIMEOUT=30s \
  -e RATE_LIMIT_REQUESTS=1000 \
  -e RATE_LIMIT_WINDOW=1m \
  -e LOGIN_RATE_LIMIT_REQUESTS=100 \
  -e LOGIN_RATE_LIMIT_WINDOW=5m \
  -e PASSWORD_RESET_RATE_LIMIT_REQUESTS=50 \
  -e PASSWORD_RESET_RATE_LIMIT_WINDOW=15m \
  -e IDEMPOTENCY_TTL=10m \
  -e TRACING_ENABLED=true \
  -e TRACING_ENDPOINT=otel-collector:4318 \
  -e TRACING_INSECURE=true \
  -e TRACING_SERVICE_NAME=taskflow \
  taskflow-taskflow
docker exec taskflow-mongo-1 mongosh --quiet mongodb://127.0.0.1:27017/taskflow --eval 'db.runtime_rate_limits.deleteMany({}); db.runtime_idempotency_keys.deleteMany({});'
docker run --rm --network taskflow_default -v "$PWD/scripts:/scripts" \
  -e BASE_URL=http://taskflow-highlimit:8080 \
  -e VUS=5 \
  -e DURATION=20s \
  grafana/k6:latest run /scripts/perf_smoke.js
```

Result:

- `http_req_failed=0.00%`
- `http_req_duration p(95)=14.55ms`
- `http_reqs=201`
- `iterations=100`

Notes:

- This reproduces the same `5 VU / 20s` shape that previously produced malformed HTTP responses and process restarts under Go `1.26.4`.
- After pinning to Go `1.25.11`, the crash did not reproduce.

## Conclusion

- The OTLP tracing path is stable on the pinned Go `1.25.11` runtime baseline for the current smoke profile.
- Default compose limits remain intentionally low for runtime protection, so heavier perf runs should use a dedicated high-limit environment instead of changing production-like defaults.

## Remaining Gaps

- No soak test or long-duration latency distribution yet
- No capacity curve across larger VU counts
- No CPU / memory profile capture in this baseline
