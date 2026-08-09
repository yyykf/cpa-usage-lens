---
关联: trellis:08-09-durable-cpa-queue-buffer
---

# vmrack durable usage-buffer candidate validation

## Scope

- Candidate commit: `43e8a92e70fdc0baa73b382396d505d03feea885`.
- Backend-only local candidate image:
  `ghcr.io/yyykf/cpa-usage-lens-backend:deploy-43e8a92`.
- Frontend remained on official `v0.6.0`; no database migration or environment
  key was added.
- Live CPA version: `v7.2.125`, commit `2e6b1d8`.

## Safety and deployment

- Confirmed official Lens `v0.6.0`, one backend collector, restart count `0`,
  and an empty persistent buffer before deployment.
- Saved `.env.bak.pre-durable-buffer-43e8a92-20260809-142529`.
- Stopped the old backend, set `COLLECTOR_ENABLED=false`, and started the
  candidate read-only before re-enabling one collector.
- The first Compose recreation retained `v0.6.0` because `sudo` removed the
  temporary `CUL_VERSION`; the collector was disabled, so no queue item was
  consumed by that short-lived process. Re-ran with
  `sudo env CUL_VERSION=deploy-43e8a92` and verified the candidate image before
  enabling collection.
- Candidate backend started at `2026-08-09T14:26:47Z`.
- Local and vmrack `/app/server` SHA-256 both equal
  `c54a5a5562c8585be42028bf12d87d6faf3688e88ada56ca35e450e0eb2f882a`.

## Live acceptance

- Observed 19 periodic snapshots from `14:28:22Z` through `14:38:10Z`, plus
  pre/post checks covering more than 12 minutes.
- Backend remained `running`, restart count `0`, with zero candidate error lines
  matching data-loss, rejected-normalization, buffer failure, panic, or fatal
  conditions.
- `collector_state.events_ingested` advanced from `46543` before candidate
  collection to `46705` at final readback: `162` newly inserted events.
- Collector API reported a current poll/event waterline and empty `lastError`.
- Persistent buffer ended with `0` pending `.json`, `0` `.rejected`, and `0`
  `.corrupt` files; the final filesystem scan found no secret-bearing artifact.
- Overview, accounts, keys, trend, and models APIs returned HTTP `200`.
- Today's overview contained `1601` requests, all `complete`, with `0`
  unclassified, `0` inconsistent, and `0` legacy requests.
- `costCoverage=unknown` existed on official `v0.6.0` before candidate deployment
  and remained unrelated missing-price state; this collector-only change did not
  alter pricing or report behavior.

## Verification already completed locally

- `go test ./...`
- `go test -race ./...`
- `go vet ./...`
- `gofmt -l backend/internal/collector` returned no files
- frontend `npm run build`
- production/debug Compose config validation
- `git diff --check`

## Rollback

- Stop the candidate backend first.
- Restore the saved `.env` or keep `COLLECTOR_ENABLED=true` as appropriate.
- Start backend with `CUL_VERSION=v0.6.0`; no schema or data rollback is needed.
- Candidate and official images remain available locally on vmrack until formal
  release validation finishes.

## Official release and deployment

- PR [#21](https://github.com/yyykf/cpa-usage-lens/pull/21) merged as
  `d828d9fb9da1193bb5a3866729ccf4822631280c` after all CI checks passed.
- Annotated tag `v0.6.1` resolves exactly to that application merge commit.
- Release workflow
  [31319207569](https://github.com/yyykf/cpa-usage-lens/actions/runs/31319207569)
  completed successfully and published multi-architecture backend/frontend
  images plus the [v0.6.1 release](https://github.com/yyykf/cpa-usage-lens/releases/tag/v0.6.1).
- vmrack switched both services to official `v0.6.1` at
  `2026-08-09T14:46:06Z`; collector remained enabled and unique.
- Official backend `/app/server` SHA-256 remained identical to the accepted
  candidate:
  `c54a5a5562c8585be42028bf12d87d6faf3688e88ada56ca35e450e0eb2f882a`.
- Eleven official-image snapshots through `14:52:20Z` showed restart count `0`,
  empty collector error, and `0` pending/rejected/corrupt buffer files.
- `events_ingested` advanced from `46818` to `46888` during the periodic window
  and reached `46893` at final readback.
- Final overview contained `1781` requests, all accounting quality `complete`,
  with `0` unclassified, `0` inconsistent, and `0` legacy requests.
- Final backend log scan found `0` data-loss, rejected-normalization, buffer,
  write, panic, or fatal error lines; all public APIs remained healthy.
- Release jobs emitted non-blocking GitHub runner warnings that several pinned
  actions still target Node.js 20 and were forced onto Node.js 24. The release
  succeeded; action-version maintenance is outside this collector fix.
