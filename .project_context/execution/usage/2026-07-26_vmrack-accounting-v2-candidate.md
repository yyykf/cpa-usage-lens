---
关联: trellis:07-26-adopt-cpa-v2-token-accounting
---

# vmrack accounting v2 candidate deployment

## Scope

- Deployed commit `2094f4c` as local-only candidate images tagged `deploy-2094f4c`.
- Validated against the live vmrack CPA `v7.2.100` (`27fc316`).
- Did not publish candidate images to GHCR and did not create a release tag.

## Safety and migration

- Confirmed CPA usage queue retention is `600` seconds and the production collector is unique.
- Created pre-migration backup
  `/home/code4j/cpa-usage-lens/backups/20260726-1225-pre-accounting-v2-2094f4c.dump`
  with SHA-256 `10f6a80d496ddbbc42ff039e9f8da6a02c4030057241b35174e706321c23e485`.
- Stopped the old backend at `2026-07-26T04:23:46Z`, applied
  `20260726000000_add_token_accounting_v2.sql` transactionally, and started the
  candidate with `COLLECTOR_ENABLED=false` at `04:23:48Z`.
- Verified all 18 new storage fields, legacy backfill, non-negative canonical
  counters, authenticated APIs, and health before enabling the collector at `04:26:06Z`.

## Live acceptance

- Observed backend and frontend for more than 10 minutes with 17 consecutive
  health snapshots; both stayed running with zero restarts.
- Collector `events_ingested` advanced from `36606` to `36638`.
- Captured 32 real CPA accounting-v2 events, all `quality=complete`, representing
  8,879,764 total tokens with zero unclassified tokens.
- All 32 v2 records satisfied the six-bucket sum invariant. Their canonical totals
  included 28,514 uncached input, 8,844,288 cache read, 0 cache write, 5,020
  non-reasoning output, and 1,942 reasoning tokens.
- Daily rollup incorporated 30 new v2 requests by the final readback; the remaining
  two were newer than the last completed rollup window.
- Authenticated 7-day overview returned `costCoverage=complete`; six canonical
  segments exactly equalled 505,942,878 total tokens. Accounts, keys, trend,
  models, and collector APIs all returned HTTP 200.
- Backend logs contained no error, failure, or panic lines. The persistent buffer
  contained no pending files.

## Rollback

- `.env.bak.pre-accounting-v2-2094f4c`, the v0.5.0 images, and the database dump
  remain on vmrack.
- The migration is additive and leaves primary keys unchanged. For rollback, stop
  the candidate collector first, restore the previous environment and images, and
  restore the dump only if the additive schema/data changes themselves must be removed.
