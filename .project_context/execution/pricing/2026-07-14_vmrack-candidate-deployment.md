# vmrack candidate pricing deployment

## Scope

- Deployed commit `49d9d27` from `codex/fix-codex-usage-pricing` to the existing
  vmrack production Compose stack as the non-release tag `deploy-49d9d27`.
- Kept the branch unmerged and did not publish a GHCR release tag.
- Confirmed CPA queue retention was `600` seconds before stopping the old collector.

## Deployment

- Built `linux/amd64` backend and frontend images locally and loaded them directly
  on vmrack.
- Created the pre-migration public-schema backup at
  `/home/code4j/cpa-usage-lens/backups/20260714-1430-pre-long-context-49d9d27.dump`.
- Stopped the old backend at `2026-07-14T14:33:48Z`, applied
  `20260712220233_add_long_context_pricing.sql` transactionally, and started the
  candidate backend with `COLLECTOR_ENABLED=false` at `14:33:50Z`.
- Verified the candidate schema, prices, health endpoint, and frontend before
  re-enabling the collector at `14:36:33Z`.

## Verification

- Backend and frontend health checks returned `ok`; both candidate containers
  remained `running` with restart count `0` throughout a 10-minute observation.
- Public dashboard traffic through `https://cpa-usage.llmisland.com/` returned HTTP
  200 for collector, overview, model, account, key, and trend APIs.
- `daily_account_usage` uses the five-column primary key
  `(usage_date, source, model, key_fingerprint, long_context)`.
- GPT-5.4, GPT-5.5, and GPT-5.6 Luna/Sol/Terra have a `272000` long-context threshold;
  GPT-5.6 high-tier cache-write prices were populated.
- During observation, `request_events_hot` grew from `12984` to `13080` and
  `collector_state.events_ingested` grew from `26186` to `26282`; the final daily
  request total also reached `26282`.
- Long-context rollup produced `27` rows covering `1235` requests, `372916011`
  input tokens, and `981598` output tokens.
- The persistent backend buffer was empty after collection and rollup completed.
- No new collector error was recorded; the retained `last_error_at` remained the
  historical `2026-06-23T10:13:41Z` DNS incident.

## Post-validation data cleanup

- Removed the temporary `grok-4.5` trial data at the user's request before the
  formal release: `32` hot events and `24` daily rows representing `32` requests,
  `116528` input tokens, and `1067` output tokens.
- Deleted hot and daily rows in one short transaction while holding write-safe
  table locks so the concurrent collector and rollup could not recreate a partial
  result.
- Verified both tables contain zero `grok-4.5` rows after later rollup cycles and
  that every remaining daily row with token usage resolves to a model price.
- The separate `grok-4.5` to `xai/grok-4.5` LiteLLM alias compatibility is
  intentionally deferred and was not added to this release.

## Rollback

- The previous `v0.4.0` images and `.env.bak.pre-long-context-49d9d27` remain on
  vmrack.
- Do not start the old binary against the migrated five-column primary key.
  Rollback requires stopping the candidate collector and restoring the database
  backup together with the previous environment/image, or applying a forward fix.
