---
关联: trellis:07-26-adopt-cpa-v2-token-accounting
---

# CPA v2 token accounting

## Implemented

- Parsed and validated CPA v7.2.97 accounting v2 payloads with overflow-safe invariants.
- Preserved legacy CPA compatibility while quarantining malformed v2 payloads as inconsistent.
- Added canonical token buckets and quality counters to hot and daily storage.
- Calculated costs from reliable classified buckets and exposed complete, partial, or unknown coverage.
- Updated the dashboard to show six mutually exclusive token buckets and partial-statistics warnings.

## Verification

- `go test ./...`
- `npm run build`
- PostgreSQL 17 integration test with the new migration applied twice

## Rollback

The migration is additive and does not alter primary keys. Stop the new collector before running an old binary.
Old binaries ignore the new columns, but daily rows produced after rollback will not carry accounting-v2 quality data.
