# Backend Quality Gates

## Trigger

Apply this spec to every backend, migration, runtime-config, or backend API DTO
change before commit and review.

## Required Design Practices

- Keep changes inside the existing ownership boundary; use a consumer-owned
  small interface when a dependency needs substitution.
- Keep functions focused on one stage: transport, validation, persistence,
  aggregation, or presentation. Do not duplicate token/accounting rules across
  stages.
- Preserve context cancellation through DB, HTTP, schedulers, and collector
  operations.
- Time-dependent logic SHOULD receive a time value or location so boundary
  behavior is deterministic in tests.
- JSON fields that are part of the frontend contract MUST remain present at
  zero; do not add `omitempty` to required numeric/count/quality fields.
- A known safety defect MUST be fixed or explicitly tracked; specs must not be
  weakened to describe data loss as acceptable behavior.

Evidence: small store interfaces in API/collector/rollup, table-driven config and
time tests, DTO zero-value tests in `internal/model/types_test.go`, and the
collector replay test suite.

## Test Requirements

| Change | Minimum proof |
| --- | --- |
| pure calculation/parser | focused unit tests for base, edge, invalid cases |
| bug fix | regression test that would fail before the fix |
| handler/auth behavior | status/body plus collaborator-call assertions |
| goroutine, buffer, cache, shared state | `-race` test and shutdown/recovery path |
| DTO field | JSON presence test and frontend production build |
| SQL column or scan | query/scan update plus PostgreSQL integration test |
| migration | apply twice; assert schema, defaults, existing and new row behavior |
| collector order/safety | persisted artifact state, secret exclusion, DB outcome |

- Tests SHOULD live beside production code and use local fakes that implement
  the package's small interface.
- Do not make unit tests depend on live CPA, Supabase, LiteLLM, vmrack, or wall
  clock timing.
- PostgreSQL integration tests use build tag `integration` and
  `TEST_DATABASE_URL`; the database MUST be disposable.

## CI-Equivalent Gate

From `backend/`:

```bash
go build ./...
go vet ./...
unformatted="$(gofmt -l .)"; test -z "$unformatted"
go test ./... -race
```

This mirrors `.github/workflows/ci.yml`. Run the integration test when its
trigger applies:

```bash
: "${TEST_DATABASE_URL:?set TEST_DATABASE_URL to a disposable PostgreSQL URL}"
go test -tags=integration ./internal/db
```

For a JSON DTO or token-accounting change, also run `npm run build` from
`frontend/`.

## Avoid

- Adding behavior with only happy-path tests.
- Sleeping in tests when a channel, injected clock/time, or explicit call can
  prove the result.
- Skipping `rows.Err`, batch result consumption, error return, or context.
- Suppressing `go vet`, race, or formatting failures.
- Using a production DB or destructive CPA usage queue as a test fixture.
- Documenting a verification command that CI or the repository cannot run.

## Review Checklist

- [ ] Ownership and dependency direction still match
      [Directory Structure](./directory-structure.md).
- [ ] Data lineage and conflict keys still match
      [Database](./database-guidelines.md).
- [ ] Error and logs expose enough safe context without secrets.
- [ ] Cross-layer fields were updated and tested in every consumer.
- [ ] CI-equivalent commands pass and `git diff --check` is clean.
