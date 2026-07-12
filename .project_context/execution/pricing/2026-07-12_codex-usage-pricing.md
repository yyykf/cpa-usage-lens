---
关联: trellis:07-12-fix-codex-usage-pricing
---

# Codex usage pricing compatibility

## Implemented

- Normalized CPA v7.2.67 Codex/Claude cache aliases while retaining compatibility
  with older payloads.
- Added provider-aware input/cache splitting and OpenAI-only reasoning deduplication.
- Parsed LiteLLM provider, model threshold, and four long-context prices.
- Added `long_context` as a daily rollup dimension and rebuilt retained hot days
  atomically so metadata changes cannot leave stale tier rows.
- Added provider-aware display fields for uncached input and canonical cache reads.
- Documented the breaking Lens schema/binary deployment boundary and rollback limits.

## Verification

- `go test ./...`
- `go vet ./...`
- `gofmt -l .`
- `go test -tags=integration ./internal/db -run '^TestLongContextMigrationAndRollup$'`
  against a disposable PostgreSQL 17 container
- `npm run build`

## Rollback

Stop the new collector before rollback. An old Lens binary cannot run against the
five-column daily primary key. Prefer a forward fix or restore a pre-migration
database backup; a schema-only downgrade must first merge base/long rows sharing
the previous four-column key.
