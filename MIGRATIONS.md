# TaskFlow Migration Discipline

## Scope

TaskFlow uses versioned, forward-only Mongo migrations through `cmd/migrate` and the shared runner in `internal/migrations`.

## Rules

1. Every migration version must be unique, lexically sortable, and append-only.
2. Every migration must be additive or backward-compatible for at least one rollout window.
3. Destructive schema changes must be split into multiple releases:
   - release A: add new fields/indexes and dual-read/dual-write if needed
   - release B: remove deprecated paths only after old binaries are drained
4. Runtime startup may apply pending migrations, but production rollout should still run `taskflow-migrate` explicitly before shifting traffic.
5. CI must keep a migrate smoke step plus Mongo-backed integration coverage.

## Authoring Checklist

- add a new entry to `internal/migrations.Definitions`
- give it a unique version and explicit description
- keep the list sorted
- add or update tests in `internal/migrations/migrations_test.go`
- document any rollout assumptions in `DEPLOYMENT.md` if operator behavior changes

## Rollback Rule

Current migrations are treated as forward-only. Rollback means reverting application binaries while keeping additive schema/index changes in place.
