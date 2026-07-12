# LiteLLM threshold-pricing research

## Current price metadata

Lens downloads LiteLLM's live price JSON:

<https://raw.githubusercontent.com/BerriAI/litellm/main/model_prices_and_context_window.json>

The current GPT-5 price records expose four high-context fields relevant to
this task:

```text
input_cost_per_token_above_272k_tokens
output_cost_per_token_above_272k_tokens
cache_read_input_token_cost_above_272k_tokens
cache_creation_input_token_cost_above_272k_tokens
```

For example, the current default-tier `gpt-5.6-sol` values are:

| Token category | At or below 272K | Above 272K |
| --- | ---: | ---: |
| Uncached input | $5.00 / 1M | $10.00 / 1M |
| Cache read | $0.50 / 1M | $1.00 / 1M |
| Cache write | $6.25 / 1M | $12.50 / 1M |
| Output | $30.00 / 1M | $45.00 / 1M |

GPT-5.4, GPT-5.5, and GPT-5.6 Sol/Terra/Luna all currently carry the same
`above_272k_tokens` boundary, with model-specific prices.

GPT-5.4 mini/nano do not publish an `above_272k_tokens` price. Their official
maximum input is exactly 272,000 tokens, so base-price fallback is the expected
behavior for those models.

## LiteLLM calculator behavior

LiteLLM's current `generic_cost_per_token` path:

1. reads threshold-bearing keys from model metadata;
2. compares the single request's `usage.prompt_tokens` with each threshold;
3. if the threshold is crossed, replaces input, output, cache-read, and
   cache-write unit prices for all of that request's tokens;
4. supports threshold names other than 272K and chooses the highest crossed
   threshold when multiple thresholds exist.

Source:

- <https://github.com/BerriAI/litellm/blob/main/litellm/litellm_core_utils/llm_cost_calc/utils.py>

LiteLLM also explicitly treats OpenAI-compatible `prompt_tokens` as including
both cached and cache-creation tokens, calculating ordinary input as:

```text
prompt_tokens - cached_tokens - cache_creation_tokens
```

## Lens gap

Lens does not call LiteLLM's Python cost calculator. It only downloads the JSON
and calculates cost in Go.

Current gaps:

- `backend/internal/pricing/litellm.go` parses only the four baseline price
  fields and drops all `above_*_tokens` fields.
- `backend/internal/pricing/cost.go` accepts only one price tier.
- `backend/internal/db/rollup.go` aggregates all requests for a
  day/account/model/key into one row, so a later query cannot distinguish one
  300K request from many small requests with the same daily total.

Therefore the long-context classification must happen while per-request hot
rows still exist, and the chosen price tier must survive daily rollup.

## Design implication

For the current MVP, each model needs at most one threshold and two price tiers.
The threshold should be parsed from LiteLLM metadata into `model_prices`; the
daily aggregate must retain either:

- a `long_context` row dimension, or
- a mirrored set of long-context token buckets in the existing row.

Hard-coding `272000` globally would work for today's target models but would
throw away information already present in the price source and make later
models harder to support.
