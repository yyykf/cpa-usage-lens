# Cost Calculation Contract

> Read this before changing CPA token parsing, LiteLLM price fields, daily rollup,
> report DTOs, or `internal/pricing/cost.go`.

## Scenario: Provider-aware cache and long-context pricing

### 1. Scope / Trigger

- Trigger: any change to CPA token aliases, provider semantics, reasoning tokens,
  LiteLLM price parsing, `daily_account_usage`, or token-composition display.
- Cost is computed at query time; fixed cost must never be persisted.
- The current MVP supports the default service tier and one threshold per model.

### 2. Signatures

CPA queue input:

```text
tokens.input_tokens
tokens.output_tokens
tokens.reasoning_tokens
tokens.cached_tokens
tokens.cache_read_tokens
tokens.cache_creation_tokens
```

Price contract (`model_prices` / `model.ModelPrice`):

```text
provider
input/output/cache_read/cache_creation base prices
long_context_threshold_tokens
input/output/cache_read/cache_creation long-context prices
```

Daily primary key:

```text
(usage_date, source, model, key_fingerprint, long_context)
```

Report display fields retain raw counts and add canonical fields:

```text
uncachedInputTokens
canonicalCacheReadTokens
```

### 3. Contracts

#### OpenAI / Codex

CPA v7.2.67 copies the same cache read into both `cached_tokens` and
`cache_read_tokens`, and maps OpenAI `cache_write_tokens` to
`cache_creation_tokens`. Canonical billing is:

```text
cache_read = max(cached_tokens, cache_read_tokens)
cache_write = cache_creation_tokens
uncached_input = max(input_tokens - cache_read - cache_write, 0)
```

Both read and write may be non-zero in one request. Never add the two cache-read
aliases together.

#### Anthropic / Claude

Claude input excludes its separately reported cache tokens:

```text
uncached_input = input_tokens
cache_read = cache_read_tokens
cache_write = cache_creation_tokens
```

Ignore CPA's Claude `cached_tokens` compatibility alias, including the
creation-only fallback introduced in v7.2.67.

#### Reasoning

- OpenAI `reasoning_tokens` is a subset of `output_tokens`; charge output once.
- Non-OpenAI providers retain the legacy independent reasoning charge unless
  their provider contract proves reasoning is included in output.

#### Long context

- Threshold comes from LiteLLM metadata; never hard-code 272000 globally.
- Classification is strict: `input_tokens > threshold`.
- Crossing the threshold switches input, output, cache-read, and cache-write
  prices for the full request; this is not marginal pricing.
- Classify while request-level hot detail exists and persist the boolean tier in
  daily rollup. Never infer it from a daily token sum.
- If LiteLLM publishes multiple thresholds for one model, leave the boolean-tier
  metadata unset rather than silently collapsing multiple tiers into one.

### 4. Validation & Error Matrix

| Condition | Required behavior |
| --- | --- |
| Old CPA omits cache write | Decode as zero; ingestion continues |
| CPA v7.2.67 duplicates cache read aliases | Canonicalize and bill/display once |
| Cache read + write exceeds OpenAI input | Clamp uncached input to zero |
| Required base/high input or output price is missing | Cost is unknown, never zero |
| Model threshold is missing | Rollup uses base bucket |
| Price refresh later adds/changes threshold | Rebuild retained hot days and remove stale tier rows |
| Existing daily row during migration | Preserve as `long_context=false` |
| Old Lens binary against five-column daily key | Rollup fails; zero-downtime upgrade is unsupported |

### 5. Good / Base / Bad Cases

- Good: OpenAI `input=100, cached=30, cache_read=30, creation=40` bills
  30 ordinary input, 30 cache read, and 40 cache write.
- Base: a model without threshold fields keeps existing base-price behavior.
- Good: `input=272000` remains base while `input=272001` uses the high tier.
- Bad: `input * base_price + cached * read_price + creation * write_price` for
  OpenAI; this charges cached input more than once.
- Bad: subtracting cache creation from Claude input; Claude input is already
  independent.
- Bad: classifying long context from `sum(input_tokens)` after daily aggregation.

### 6. Tests Required

- Pricing unit tests:
  - CPA v7.2.67 OpenAI aliases + cache write.
  - OpenAI cache-write-only.
  - Claude duplicated aliases remain independent.
  - OpenAI reasoning subset and non-OpenAI reasoning regression.
  - base versus long-context whole-request prices.
- LiteLLM parsing tests: provider, `272k -> 272000`, four high prices, no-threshold
  fallback.
- Collector tests: Codex and Claude alias normalization.
- Report tests: mixed base/long rows price separately but aggregate totals;
  canonical display fields are provider-aware.
- PostgreSQL integration test: idempotent migration, existing-row default,
  strict threshold boundary, two-bucket rollup, price round-trip, and stale-bucket
  removal after threshold change.
- Frontend production build must type-check the new DTO fields.

### 7. Wrong vs Correct

#### Wrong

```go
cost := input*inputPrice + cached*cacheReadPrice + cacheWrite*cacheWritePrice
cost += output*outputPrice + reasoning*outputPrice
```

This double-charges OpenAI cache tokens and reasoning.

#### Correct

```go
split := pricing.SplitInputTokens(tokens, price.Provider)
cost := split.Uncached*inputPrice + split.CacheRead*cacheReadPrice +
    split.CacheCreation*cacheWritePrice
cost += output * outputPrice
// Add reasoning separately only for providers whose output excludes it.
```

Select the base or long-context price set before applying this shared split.

## Scenario: CPA token accounting v2 quality

### 1. Scope / Trigger

- Trigger: CPA `accounting_version`, canonical buckets, accounting quality, cost coverage, or token-composition changes.
- CPA v7.2.97+ canonical accounting crosses collector, hot storage, daily rollup, report DTOs, and frontend display.

### 2. Signatures

Queue fields:

```text
accounting_version: 2
token_breakdown.schema_version: 2
token_breakdown.quality: complete | unclassified | inconsistent
token_breakdown.input: total_tokens, uncached_tokens, cache_read_tokens, cache_write_tokens
token_breakdown.output: total_tokens, non_reasoning_tokens, reasoning_tokens
token_breakdown.unclassified_tokens
```

Report fields include `nonReasoningOutputTokens`, `unclassifiedTokens`,
`costCoverage`, and request counts for all three quality values plus legacy.

### 3. Contracts

- Validate v2 once at the collector boundary; downstream layers consume the validated canonical shape.
- Input children equal input total, output children equal output total, and input + output + unclassified equal total.
- All counters are non-negative and summation must reject int64 overflow.
- A malformed declared-v2 record is `inconsistent`; never downgrade it to a legacy complete record.
- Missing v2 fields from an old CPA use the existing provider-aware legacy fallback.
- Classified buckets may be priced; unclassified tokens are retained but never assigned a guessed price.
- A mixed range returns the classified cost with `costCoverage=partial`; only a wholly unpriceable range is `unknown`.

### 4. Validation & Error Matrix

| Condition | Required behavior |
| --- | --- |
| Valid complete v2 | Store and price all five classified buckets |
| Valid partial/unclassified v2 | Store total and unclassified; price only classified buckets |
| Contradictory or overflowing v2 | Mark inconsistent and quarantine total as unclassified |
| `accounting_version=2` without breakdown | Mark inconsistent |
| Legacy payload without version | Preserve provider-aware fallback and count as legacy |
| One bad record mixed with good records | Keep known cost and mark the aggregate partial |

### 5. Good / Base / Bad Cases

- Good: `input=100 (60+30+10)`, `output=20 (5+15)`, total `120`, quality complete.
- Base: an old CPA payload still produces the same legacy cost and remains visible as legacy coverage.
- Good: classified `100` plus unclassified `20` displays total `120` and a partial cost for the classified portion.
- Bad: one inconsistent request turns an otherwise valid 30-day cost into null.
- Bad: invalid v2 silently falls back to field-shape guessing.

### 6. Tests Required

- Collector fixtures for complete, partial/unclassified, malformed, missing, negative, and overflowing v2 payloads.
- Pricing/report tests proving partial classified cost remains visible and all-unclassified cost is unknown.
- DTO tests proving canonical and coverage fields remain present at zero.
- PostgreSQL integration test applies the additive migration twice and verifies hot-to-daily canonical lineage.
- Frontend production build type-checks all six mutually exclusive display buckets.

### 7. Wrong vs Correct

#### Wrong

```go
if !breakdown.Valid() {
    return legacyProviderGuess(raw.Tokens)
}
```

This makes a malformed declared-v2 record look trustworthy.

#### Correct

```go
if declaredV2 && !valid {
    return inconsistentWithUnclassifiedTotal(authoritativeTotal)
}
if !declaredV2 {
    return legacyProviderFallback(raw.Tokens)
}
```
