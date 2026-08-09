# Cross-Layer Thinking Guide

## Trigger

Use when a change touches two or more of CPA transport, replay, Go model,
PostgreSQL, rollup, pricing/report, HTTP DTO, TypeScript DTO, or UI display.

## Map the Real Data Flow

```text
CPA destructive queue
  -> sanitized durable replay
  -> typed validation and canonical accounting
  -> request_events_hot
  -> daily_account_usage
  -> report plus query-time prices
  -> Go JSON DTO
  -> TypeScript DTO/default
  -> dashboard views
```

For deployment/schema work, also map:

```text
Git commit/tag -> release workflow -> backend/frontend images -> Compose/nginx
```

## Boundary Questions

### Source and ingestion

- [ ] Is the source destructive, retryable, or reconstructable?
- [ ] Are secrets removed before persistence/logging?
- [ ] Which layer validates once, and what happens to invalid siblings?
- [ ] What identity makes replay idempotent without collapsing real events?

### Storage and rollup

- [ ] Is the field a request-level fact, derived daily value, or query-time value?
- [ ] Did hot insert, migration, rollup select/upsert, daily query/scan, and Go
      models all change together?
- [ ] Can retained daily data be rebuilt atomically from hot data?
- [ ] Does a failure stop cleanup before data is lost?

### Report and API

- [ ] Which layer owns provider/accounting/cost semantics?
- [ ] Are totals, canonical buckets, quality counts, and cost coverage consistent?
- [ ] Are JSON names, zero-field presence, nullability, and stable error messages
      explicit?
- [ ] Does one bad row produce partial, unknown, rejection, or whole-request
      failure, and why?

### Frontend

- [ ] Did `types.ts`, default objects, every display, sort, chart, and formatter
      receive the same contract?
- [ ] Are loading, empty, zero, null, partial, and unknown distinct?
- [ ] Can an old response overwrite a newer state after the change?
- [ ] Does the UI reuse canonical data instead of re-deriving backend semantics?

### Deployment

- [ ] Is binary/schema order backward compatible? If not, is downtime order and
      rollback limit explicit?
- [ ] Does production topology preserve one collector and private backend port?
- [ ] Can validation avoid calling the destructive usage queue?
- [ ] Are release artifact, candidate, and deployed artifact identities proven?

## Required Artifacts by Change

| Change | Minimum synchronized evidence |
| --- | --- |
| DTO field | Go JSON test, TypeScript type/default/consumer, frontend build |
| persisted usage field | migration, hot insert, rollup, query/scan, models, integration test |
| accounting/cost rule | collector/pricing/report tests and all UI coverage states |
| migration key | ADR, idempotent integration test, deployment/rollback contract |
| queue behavior | replay-order, secret-exclusion, recovery, idempotency tests |
| deployment topology | Compose/nginx validation and released artifact provenance |

## Stop Conditions

Do not implement until resolved when:

- one field has two proposed owners;
- a total cannot be reconciled across layers;
- a destructive read precedes durable sanitized evidence without explicit
  accepted loss;
- null/unknown would be converted to zero;
- a migration cannot be applied safely with known rollout order;
- verification requires production mutation or destructive queue inspection.

Then load the normative backend/frontend spec selected by their indexes and
record architecture rationale in an ADR if the current design changes.
