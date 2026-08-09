# PostgreSQL and Data-Lineage Contract

## Trigger

Apply this spec to SQL, pgx calls, table/index definitions, migrations, rollup,
retention, persisted token fields, or price-cache storage.

## Storage Roles

```text
request_events_hot   request-level ingested facts within retention
daily_account_usage  derived daily aggregate, rebuilt from retained hot facts
model_prices         refreshable price metadata/cache
collector_state      observable collector health and counters
```

- Hot rows are the retained source of truth for request-level facts. Daily rows
  MUST remain reproducible from hot rows for the retained window.
- Cost MUST NOT be persisted. Reports combine usage with current `model_prices`
  at query time.
- The configured `TIMEZONE` defines daily boundaries. Query ranges use
  `[startDate, endDate)`; rollup rebuild parameters use the explicitly documented
  inclusive date endpoints.

Evidence: `backend/internal/db/events.go`, `queries.go`, `rollup.go`,
`backend/internal/report/report.go`, and `backend/internal/timeutil/timeutil.go`.

## Query Pattern

- Use `pgx/v5` and `pgxpool`; do not introduce an ORM without a new architecture
  decision and repository-wide migration plan.
- All values MUST use positional parameters. Never build SQL by interpolating
  request, model, date, source, or key data.
- Multi-row ingestion SHOULD use `pgx.Batch`; every queued result MUST be
  consumed and the batch result closed.
- Every `Query` MUST `defer rows.Close()` and return `rows.Err()` after scanning.
- The `SELECT` column list and `rows.Scan` destination list are one contract.
  Change and review them together, in the same order.
- Empty input SHOULD return before opening a batch or transaction.

## Identity and Idempotency

- `request_events_hot` identity is
  `(request_id, event_ts, total_tokens)`. `InsertEvents` MUST retain
  `ON CONFLICT ... DO NOTHING` so replay is idempotent while WebSocket turns
  sharing one `request_id` remain distinct.
- `daily_account_usage` identity includes
  `(usage_date, source, model, key_fingerprint, long_context)`.
- A conflict-key change is breaking across binary and schema. Follow
  [Deployment Guidelines](./deployment-guidelines.md) and record a new ADR.

Rationale: ADR [0001](../../../.project_context/design/decisions/0001-usage-hot-composite-pk.md)
and [0002](../../../.project_context/design/decisions/0002-long-context-pricing-dimension.md).

## Rollup and Retention

- Rebuild of a retained window MUST happen in one transaction: delete target
  daily rows, aggregate current hot rows, upsert all daily columns, commit.
- New persisted usage/accounting fields MUST be added to hot insert, daily
  schema, rollup `INSERT/SELECT/UPDATE`, query `SELECT/Scan`, Go models, and the
  integration test as one cross-layer change.
- Scheduler order is mandatory: roll up the full retained window first, then
  delete hot rows older than the cutoff.
- If rollup fails, cleanup MUST be skipped. A cleanup error MUST be surfaced.
- Long-context classification MUST occur while request-level hot data exists;
  never infer a request price tier from an aggregated daily token sum.

## Migration Contract

- Add a new timestamp-prefixed SQL file under `supabase/migrations/`. Released
  migration files MUST NOT be edited or reordered.
- Migrations MUST be safe to run twice. Use guarded DDL and explicit backfill
  semantics; do not rely on a pristine database.
- Before replacing a primary key or unique constraint, inspect the real column
  set and prove old/new binary compatibility or document required downtime.
- Additive columns MUST define how existing rows are interpreted. Historical
  reconstruction MUST NOT be guessed when source detail no longer exists.
- Migration tests MUST apply the migration twice against disposable PostgreSQL
  and assert schema plus behavior, not only successful execution.

## Avoid

- Persisting calculated USD cost.
- Direct writes to daily aggregates from the collector or API.
- Querying with `<= endDate` when the public range contract is half-open.
- Updating only a SQL `SELECT` or only its `Scan` list.
- Deleting retained hot data after a failed or partial rollup.
- Editing a released migration to make a new checkout pass.

## Verify

```bash
go test ./... -race
: "${TEST_DATABASE_URL:?set TEST_DATABASE_URL to a disposable PostgreSQL URL}"
go test -tags=integration ./internal/db
```

For any migration, run the integration test twice against a disposable database
and inspect the final primary keys, indexes, defaults, and row-level result.
