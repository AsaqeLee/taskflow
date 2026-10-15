# Performance Check: Go 1.26.9 upgrade (2026-10-15)

## Why

`performance-baseline-2026-06-12.md` records malformed HTTP responses and process restarts under Go `1.26.4` (5 VU / 20 s, OTLP tracing on), which led to the Go 1.25 pin. Go 1.25.13 is now flagged by govulncheck (GO-2026-6599, GO-2026-6600 and other stdlib issues, fixed only in 1.26.9), so the upgrade was re-validated before landing.

## Environment

- Box: Debian 13, 8 vCPU, 15 GiB RAM, Docker 26.1.5 (vfs storage driver), all containers on the host network
  (bridge networking between containers was not usable on this box, so the compose stack itself was not used)
- App image built from the repository `Dockerfile`; binary toolchain verified with `go version`
- MongoDB `mongo:7` single-node replica set, OTLP collector `otel/opentelemetry-collector-contrib:0.130.1` with `deploy/otel-collector.yaml`
- App env: same as run 2 of the June baseline (`TRACING_ENABLED=true`, `TRACING_ENDPOINT=127.0.0.1:4318`), but with `RATE_LIMIT_REQUESTS=100000` and `LOGIN_RATE_LIMIT_REQUESTS=1000` so throttling does not mask errors
- k6 `grafana/k6:latest`, `scripts/perf_smoke.js`, user `u_alice` from `scripts/users.example.json`
- Three runs per image, sequentially on one container: 5 VU / 20 s (the June crash profile), 10 VU / 60 s, 20 VU / 120 s

## Results

| Image | Run | http_req_failed | http_reqs | iterations | avg | med | p90 | p95 | max |
|---|---|---|---|---|---|---|---|---|---|
| go1.25.13 | 5 VU / 20s | 0.00% | 201 | 100 | 5.73ms | 3.51ms | 5.77ms | 9.44ms | 183.01ms |
| go1.25.13 | 10 VU / 60s | 0.00% | 1201 | 600 | 7.35ms | 4.09ms | 7.04ms | 10.75ms | 122.86ms |
| go1.25.13 | 20 VU / 120s | 0.00% | 4761 | 2380 | 6.61ms | 3.94ms | 9.07ms | 13.82ms | 181.35ms |
| go1.26.9 | 5 VU / 20s | 0.00% | 201 | 100 | 8.15ms | 3.64ms | 7.83ms | 63.84ms | 94.87ms |
| go1.26.9 | 10 VU / 60s | 0.00% | 1201 | 600 | 5.32ms | 4.18ms | 7.71ms | 10.52ms | 78.53ms |
| go1.26.9 | 20 VU / 120s | 0.00% | 4721 | 2360 | 9.24ms | 4.43ms | 10.31ms | 46.26ms | 168.16ms |
| go1.26.4 | 5 VU / 20s | 0.00% | 201 | 100 | 4.48ms | 3.6ms | 5.69ms | 6.87ms | 73.64ms |
| go1.26.4 | 10 VU / 60s | 0.00% | 1201 | 600 | 5.63ms | 4.02ms | 7.56ms | 14.59ms | 74.07ms |
| go1.26.4 | 20 VU / 120s | 0.00% | 4761 | 2380 | 5.78ms | 3.66ms | 8.45ms | 11.53ms | 143.29ms |

All three images: container `RestartCount=0` and still running after all runs; 0 app log lines matching `panic|fatal|malformed|unsolicited|superfluous`; 0 k6 log lines matching `malformed|EOF|connection reset|refused`; all k6 checks passed; spans received by the collector (debug exporter).

`go1.26.4` here is the **current** code built with Go 1.26.4 (control). It did not reproduce the June failure either, so this setup does not reproduce the original bug. Possible reasons: the app changes landed in the same commit as the pin (`d0206e4`, e.g. `internal/middleware/request_runtime.go`), or the compose bridge network / resource setup from June. The pre-`d0206e4` image could not be compared directly (it has no bootstrap binary, so k6 login failed).

## Conclusion

- Go 1.26.9 shows no restarts and no malformed responses or request failures under the June crash profile or the 3–6× longer runs at 2–4× the VUs, and median latency is within 0.5 ms of 1.25.13 in every run; p95 varied run to run on this shared box (1.26.9 had two runs with p95 of 46–64 ms vs 9–14 ms for 1.25.13; medians and max were similar), so p95 differences here are not conclusive.
- Because the original failure was not reproduced even on 1.26.4, re-run this profile on the real compose stack after deploying.
