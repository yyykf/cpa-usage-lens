# vmrack spec-only deployment reconcile

## Scope

- Trigger: follow-up request to deploy after the project-spec bootstrap merged.
- Merged main head: `f00917f7774ff8ff8c0b6a86bfd201436133d38f`.
- Latest formal application release: `v0.6.1`, application merge
  `d828d9fb9da1193bb5a3866729ccf4822631280c`.
- `v0.6.1..origin/main` contains only project context, Trellis specs, and task
  bookkeeping. Backend, frontend, schema, Compose, environment contract, and
  release workflow have no diff.

## Reconcile decision

- vmrack already ran backend/frontend `v0.6.1` with restart count `0` and the
  single collector enabled.
- vmrack intentionally uses a production Compose override behind Nginx Proxy
  Manager on `proxy-net`; it exposes no host port. The frontend served its HTML
  successfully inside the container.
- Pulled both official `v0.6.1` images and compared image IDs with the running
  containers. Both matched.
- Backend `/app/server` SHA-256 remained
  `c54a5a5562c8585be42028bf12d87d6faf3688e88ada56ca35e450e0eb2f882a`,
  identical to the previously accepted candidate and official release.
- Ran an idempotent `CUL_VERSION=v0.6.1 docker compose ... up -d` using the
  vmrack-specific Compose file. Neither container was recreated; container IDs
  and start timestamps stayed unchanged.

This is the correct deployment result for a spec-only main advance: reconcile
the pinned official artifact without forcing a no-value restart or publishing a
new application tag.

## Post-reconcile acceptance

- Backend/frontend status: running; restart count `0`.
- Backend health: `ok`.
- Collector: `running`, empty `lastError`.
- `eventsIngested` advanced from `47093` to `47094` during an eight-second
  observation and reached `47099` at final readback.
- Overview: `1987` requests, all `complete`; `0` unclassified, `0`
  inconsistent, and `0` legacy requests.
- Accounts, keys, trend, and models APIs returned HTTP `200`.
- Buffer: `0` pending, `0` rejected, `0` corrupt.
- Relevant backend error scan: `0` matches.
- `costCoverage=unknown` remains the previously identified missing-price state;
  accounting quality is complete.

The destructive CPA usage-queue endpoint was not called during deployment or
acceptance.

## Rollback

No runtime state changed and no rollback is required. vmrack remains pinned to
official `v0.6.1`; the preceding container instances and start timestamps were
preserved.
