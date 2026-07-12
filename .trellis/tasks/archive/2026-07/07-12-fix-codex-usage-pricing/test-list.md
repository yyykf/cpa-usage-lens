# Canon TDD test list

## Cache and output billing

- [x] CPA v7.2.67 Codex cache read appears in both `cached_tokens` and
  `cache_read_tokens` but is billed once.
- [x] GPT-5.6 cache-write-only requests subtract the write from ordinary input
  and bill it at the cache-creation rate.
- [x] CPA before v7.2.56 with no cache-write field remains compatible.
- [x] Claude input remains independent from duplicated/fallback cache aliases.
- [x] Reasoning tokens already contained in output are not billed twice.
- [x] Inconsistent cache totals clamp ordinary input to zero.

## LiteLLM metadata and tier selection

- [x] Parse provider, one threshold, and all four above-threshold prices.
- [x] Parse `272k` as 272000 and preserve models without threshold metadata.
- [x] Exactly-at-threshold usage remains base tier; above-threshold usage uses
  high input/output/cache-read/cache-write prices for the full request.
- [x] A long-context row with a missing required high-tier price is unknown.

## Collection and display normalization

- [x] Codex v7.2.67 aliases normalize to one canonical cache-read representation.
- [x] Codex write-only and old-CPA shapes normalize without inventing tokens.
- [x] Claude read/write counters stay independent from ordinary input.
- [x] Frontend token composition subtracts OpenAI cache writes and never shows
  duplicated cache reads.

## Storage, rollup, and reporting

- [x] Migration adds model threshold/high prices and idempotently widens the
  daily primary key with `long_context` while preserving old rows as false.
- [x] Rollup classifies every request with strict `input_tokens > threshold`.
- [x] Mixed base/long requests create two rows for one date/source/model/key.
- [x] Re-roll after price metadata changes removes the stale tier row instead
  of double-counting it.
- [x] Daily query scans `long_context`; reports calculate per-row tier cost and
  aggregate both tiers into unchanged API totals.
- [x] Models without threshold metadata remain in the base bucket.

## Deployment

- [x] English and Chinese deployment docs describe stop/migrate/deploy/restart,
  old Lens binary incompatibility, and old CPA payload compatibility.
