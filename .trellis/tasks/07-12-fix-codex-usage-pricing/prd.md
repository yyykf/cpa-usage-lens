# Fix Codex usage pricing compatibility

## Goal

Make future `cpa-usage-lens` cost calculations accurate for the user's primary
path—Codex OAuth through CLIProxyAPI—after GPT-5.6 introduced billable cache
writes and the GPT-5 family introduced whole-request long-context prices,
without breaking ingestion from older CLIProxyAPI versions or regressing
Claude-style cache accounting.

## What I already know

- The user's active CLIProxyAPI version is `v7.2.67`.
- The primary path is Codex OAuth; observed `service_tier` values are currently
  `default`.
- Historical rows will not be corrected or backfilled; correctness starts with
  future data after deployment.
- `v7.2.56` began mapping OpenAI `cache_write_tokens` into the existing queue
  field `cache_creation_tokens`; Lens does not need a new raw queue/DB field.
- `v7.2.67` copies the same OpenAI cache-read count into both `cached_tokens`
  and `cache_read_tokens`; they are aliases in this path and must be billed once.
- OpenAI `cached_tokens` and `cache_write_tokens` are independent input subsets
  and may both be non-zero in one request.
- For OpenAI-compatible usage, uncached input is
  `max(input_tokens - canonical_cache_read - cache_creation_tokens, 0)`.
- OpenAI `reasoning_tokens` is an output breakdown and is currently double
  charged by Lens because Lens charges `output_tokens` and then adds reasoning
  again.
- GPT-5.4, GPT-5.5, and GPT-5.6 Sol/Terra/Luna all currently switch the full
  request/session to high-context prices when `input_tokens > 272000`.
- GPT-5.4 mini/nano have an official maximum input of exactly 272000, so they
  cannot cross that strict boundary and remain on base prices.
- The supplied production query screenshot shows this is active traffic:
  `gpt-5.6-sol` has 264 long-context requests (maximum input 337445), and
  `gpt-5.4` has 11 (maximum input 792051).
- LiteLLM already publishes model-specific threshold and high-tier prices, but
  Lens currently ignores them and loses the per-request price tier during daily
  aggregation.

## Assumptions

- The MVP supports one long-context threshold per model and two price tiers
  (base / long context). Multiple thresholds for one model are not required now.
- Missing fields from older CLIProxyAPI payloads continue to decode as zero.
- A missing high-context price or threshold must not invent a price; the
  existing "unknown cost" behavior remains preferable to silent mispricing.

## Requirements

- Prioritize Codex OAuth / OpenAI GPT-5.4 (including mini/nano), GPT-5.5, and
  GPT-5.6 Sol/Terra/Luna on the default service tier.
- Accept CPA payloads both before and after `v7.2.56`/`v7.2.67`; absent cache
  fields remain zero and never make ingestion fail.
- Canonicalize duplicated OpenAI `cached_tokens`/`cache_read_tokens` so one
  cache read is displayed and billed once.
- Bill GPT-5.6 cache writes from CPA's existing `cache_creation_tokens` field.
- Subtract both canonical cache reads and cache writes from OpenAI total input
  before charging the uncached-input price.
- Preserve provider semantics where Claude input excludes its separately
  reported cache tokens.
- Charge OpenAI reasoning exactly once as part of `output_tokens` unless a
  future explicit reasoning-token price requires different handling.
- Parse a model-specific threshold and high-context input/output/cache-read/
  cache-write prices from LiteLLM metadata.
- Classify each hot request using its own `input_tokens` before daily rollup;
  use a strict `>` threshold comparison.
- Persist the selected base/high price tier as a `long_context boolean`
  dimension in `daily_account_usage`; include it in the daily primary key.
- Apply the selected tier to the full request, including all cache and output
  token categories.
- Continue calculating cost at query time rather than storing a fixed cost.
- Keep existing API/dashboard views aggregated across the new dimension so the
  user sees the same totals, with corrected cost.
- Migrate existing daily rows to `long_context=false`; do not reclassify or
  backfill historical rows whose hot request detail is no longer retained.
- Refresh/re-roll retained hot days when new threshold metadata becomes
  available so recent requests are not permanently left in the base bucket.

## Acceptance Criteria

- [x] A CPA `v7.2.67` Codex record with input 100, cached/read 30/30, cache
  creation 40, and output 20 charges 30 ordinary input tokens, 30 cache-read
  tokens, 40 cache-write tokens, and 20 output tokens—each exactly once.
- [x] A payload from CPA before `v7.2.56` with no cache-write field still
  ingests and calculates using the fields available to that version.
- [x] A Claude-style record whose input excludes cache tokens does not subtract
  cache read/write from its ordinary input.
- [x] Reasoning tokens contained in `output_tokens` do not add a second output
  charge.
- [x] `input_tokens == threshold` uses base prices; `input_tokens > threshold`
  uses high prices for the entire request.
- [x] Mixed short and long requests for the same date/source/model/key retain
  enough information after rollup to calculate both tiers accurately.
- [x] GPT-5.4, GPT-5.5, and GPT-5.6 Sol/Terra/Luna load their current LiteLLM
  272K threshold and high-context prices.
- [x] GPT-5.4 mini/nano remain on base prices because they have no high-context
  price metadata and cannot exceed 272K input.
- [x] Models without high-context metadata continue using existing base-price
  behavior.
- [x] Existing dashboard/API totals remain aggregated across price tiers unless
  a view explicitly needs tier details.
- [x] Existing `daily_account_usage` rows migrate to `long_context=false`
  without data loss or duplicate-key failure.
- [x] Retained hot rows are reclassified on a later rollup after previously
  missing model threshold metadata is loaded.

## Definition of Done

- Unit and integration tests cover CPA version shapes, provider cache semantics,
  reasoning, threshold boundaries, mixed-tier rollup, and price parsing.
- Backend tests, lint/static checks, and frontend checks affected by API changes
  pass.
- A new migration is idempotent and documents deployment and rollback behavior.
- Deployment documentation explains whether old Lens binaries can run against
  the migrated schema.
- An execution note records verification commands and rollback constraints.

## Out of Scope

- Historical data correction or reconstruction of unreported cache writes.
- `flex`, `priority`, fast-mode, batch, or other service-tier pricing.
- Regional/data-residency uplift, Azure-specific pricing, and other currencies.
- A generic unlimited multi-threshold pricing engine.
- New dashboard controls for service tier or long-context filtering unless
  needed to preserve existing totals.

## Research References

- [`research/openai-usage-pricing-contract.md`](research/openai-usage-pricing-contract.md)
  — official cache, reasoning, and whole-request long-context semantics.
- [`research/cpa-v7.2.67-usage-contract.md`](research/cpa-v7.2.67-usage-contract.md)
  — CPA version boundary, aliases, and old-version compatibility.
- [`research/litellm-threshold-pricing.md`](research/litellm-threshold-pricing.md)
  — threshold metadata, calculator behavior, and the current Lens gap.

## Technical Approach

- Add `long_context boolean not null default false` to
  `daily_account_usage`, then rebuild its primary key as
  `(usage_date, source, model, key_fingerprint, long_context)`.
- Parse one model-specific long-context threshold plus high-tier
  input/output/cache-read/cache-write prices from LiteLLM metadata.
- During rollup, compare every hot row's `input_tokens` with its model's
  threshold and group base and long-context requests into separate daily rows.
- Keep the current token columns unchanged; cost selects the base or high price
  set from each daily row's `long_context` value.
- Normalize OpenAI/Codex cache aliases and provider-specific input semantics
  before cost calculation so OpenAI cache read/write is subtracted from total
  input while Claude's independent cache counters remain additive.
- Remove the extra OpenAI reasoning charge because `reasoning_tokens` is
  already included in `output_tokens`.
- Aggregate both `long_context` values in existing overview, account, key,
  trend, and model queries; no new frontend control is required.

## Decision (ADR-lite)

**Context**: Daily token totals lose the per-request boundary needed to decide
whether a request crossed a whole-request pricing threshold. The aggregate must
retain the request's selected price tier without storing fixed cost.

**Decision**: Use Approach A: add `long_context boolean` as a daily row and
primary-key dimension. The model-specific threshold remains in `model_prices`;
`long_context` records whether the request crossed that model's threshold.

**Consequences**:

- A date/source/model/key may have two daily rows, one for each price tier.
- Existing API views must sum both rows, while cost is calculated per row first.
- The schema remains narrow and can later evolve from the boolean to a more
  general `pricing_tier` dimension if multiple thresholds become necessary.
- This is a breaking schema migration for old Lens binaries: their four-column
  rollup `ON CONFLICT` target will no longer match a unique constraint. Deploy
  by stopping the old Lens binary, applying the migration, and starting the new
  binary. This does not affect compatibility with old CPA payload versions.
- Existing daily history is intentionally classified as base tier; rollback
  must stop the new binary before restoring the old primary key/schema.

## Implementation Plan

1. Add failing tests for CPA version/cache shapes, reasoning-token inclusion,
   threshold boundaries, price parsing, and mixed-tier rollup.
2. Extend model-price parsing/storage and provider-aware token normalization.
3. Add the idempotent `long_context` migration and update rollup/query paths.
4. Verify backend/frontend quality gates and document deployment/rollback.

## Technical Notes

- Likely code paths: `backend/internal/collector`, `backend/internal/db/rollup.go`,
  `backend/internal/model/types.go`, `backend/internal/pricing/litellm.go`,
  `backend/internal/pricing/cost.go`, query aggregation, tests, and Supabase
  migrations.
- `request_events_hot` already contains `model`, `provider`, and per-request
  `input_tokens`, so CPA does not need to report context-window length or a
  `long_context` flag.
- Existing daily totals must not classify long context from their summed input;
  100 small requests and one large request can have the same daily sum but
  different prices.
