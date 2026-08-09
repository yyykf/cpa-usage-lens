# Backend Engineering Contract

## Applies To

Read this index before changing `backend/`, `supabase/migrations/`, backend
runtime configuration, collector storage, or backend-facing deployment files.
These documents are normative guardrails for the current repository. Architecture
rationale and decision history remain in the
[ADR directory](../../../.project_context/design/decisions/).

`MUST` and `MUST NOT` are review gates. `SHOULD` is the default and needs an
explicit reason to diverge. `MAY` identifies supported choices.

## Architecture Invariants

- `backend/cmd/server/main.go` MUST remain the composition and process-lifecycle
  boundary. Domain behavior belongs under `backend/internal/`.
- CPA events flow through collector validation into request-level hot storage.
  Daily rows are derived rollups; reports read daily rows and compute cost at
  query time.
- PostgreSQL access uses pgx and explicit SQL. The repository has no ORM layer.
- Calendar ranges use the configured timezone and `[start, end)` semantics.
- CPA usage queue consumption is destructive, single-collector, sanitized, and
  at-least-once. Never use it as a read-only inspection endpoint.

## Change-Trigger Router

| Change trigger | Required specs |
| --- | --- |
| package placement, constructor, interface, server lifecycle | [Directory Structure](./directory-structure.md), [Quality](./quality-guidelines.md) |
| SQL, pgx scan, transaction, hot/daily lineage, migration | [Database](./database-guidelines.md), [Quality](./quality-guidelines.md) |
| handler status/error, retry, recovery, partial failure | [Error Handling](./error-handling.md), [Logging](./logging-guidelines.md) |
| log message, collector state, diagnostics, secret-bearing data | [Logging](./logging-guidelines.md), [Usage Queue](./usage-queue-contract.md) |
| CPA queue transport, parsing, replay, idempotency, collector replicas | [Usage Queue](./usage-queue-contract.md), [Database](./database-guidelines.md) |
| token aliases, accounting quality, provider semantics, prices, report cost | [Cost Calculation](./cost-calculation.md), [Database](./database-guidelines.md) |
| Compose, image, nginx, ports, release or breaking schema rollout | [Deployment](./deployment-guidelines.md) |

## Pre-Development Checklist

- [ ] Map the change from source input through storage, report DTO, and consumer.
- [ ] Read every spec selected by the router; the index alone is insufficient.
- [ ] Identify the existing package that owns the behavior before adding one.
- [ ] Search for the DTO field, SQL column, env key, and business term across the
      repository before changing it.
- [ ] For DB changes, list query columns, scan destinations, rollup columns,
      conflict keys, migrations, and integration assertions together.
- [ ] For CPA queue work, prove durable-save order and secret exclusion before
      making the destructive pop call reachable.
- [ ] Define the smallest regression test that fails before the behavior change.

## Quality Check

Run from `backend/` unless noted:

```bash
gofmt -w .
go build ./...
go vet ./...
go test ./... -race
```

For migration or PostgreSQL behavior, also run against a disposable database:

```bash
: "${TEST_DATABASE_URL:?set TEST_DATABASE_URL to a disposable PostgreSQL URL}"
go test -tags=integration ./internal/db
```

Run the applicable frontend build for a Go JSON DTO change, then from repository
root run `git diff --check`. Never call the destructive CPA usage queue for a
quality check or release acceptance.
