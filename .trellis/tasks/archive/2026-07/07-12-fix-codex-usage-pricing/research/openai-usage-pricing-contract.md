# OpenAI usage and long-context pricing contract

## Scope

This note records the OpenAI contracts needed by `cpa-usage-lens` for the
Codex OAuth / GPT-5 pricing fix. It intentionally does not cover service-tier,
regional, Azure, or historical repricing.

## Prompt caching usage fields

OpenAI's prompt-caching guide defines two independent usage details:

- `cached_tokens`: input tokens read from cache.
- `cache_write_tokens`: prompt tokens written to cache, available for GPT-5.6
  and later model families and billed at 1.25 times the uncached-input rate.

They are separate subsets of the request's input tokens; one does not contain
the other. A request may read an older cached prefix and write a newer prefix in
the same request, so both values may be non-zero. For OpenAI-compatible usage,
the uncached input token count used for billing is therefore:

```text
max(input_tokens - cached_tokens - cache_write_tokens, 0)
```

This is the **uncached/baseline-price input** count, not the request's total or
"real" input count. `input_tokens` remains the request's total input count.

Source:

- <https://developers.openai.com/api/docs/guides/prompt-caching#requirements>
- <https://developers.openai.com/api/docs/guides/prompt-caching>

## Reasoning tokens

OpenAI reports `reasoning_tokens` inside output/completion token details. They
are a breakdown of `output_tokens`, not extra output on top of it. When the
model does not define a special reasoning-token rate, reasoning is already
covered by charging the full `output_tokens` at the output rate.

## Long-context rule

For all models in the current Codex OAuth MVP scope, the boundary is a strict
`input_tokens > 272000` comparison. Crossing it changes the price of the full
request/session:

- all input-side token categories use the high-context input/cache rates;
- all output tokens use the high-context output rate;
- this is not marginal pricing applied only to the tokens above 272K.

Official wording by model:

- GPT-5.4: prompts above 272K are charged at 2x input and 1.5x output for the
  full session (standard, batch, and flex).
- GPT-5.5: prompts above 272K are charged at 2x input and 1.5x output for the
  full session (standard, batch, and flex).
- GPT-5.6 Sol/Terra/Luna: prompts above 272K are charged at 2x input and 1.5x
  output for the full request; cache writes are 1.25x uncached input.

Sources:

- <https://developers.openai.com/api/docs/models/gpt-5.4>
- <https://developers.openai.com/api/docs/models/gpt-5.5>
- <https://developers.openai.com/api/docs/models/gpt-5.6-sol>
- <https://developers.openai.com/api/docs/models/gpt-5.6-terra>
- <https://developers.openai.com/api/docs/models/gpt-5.6-luna>

## Threshold scope

The 272K threshold covers the user's current GPT-5.4, GPT-5.5, and GPT-5.6
Sol/Terra/Luna traffic. It must not become a global invariant for every model or
provider. The threshold belongs to model price metadata so a later model can
use a different boundary without changing the usage schema or hard-coding a
new model list.

GPT-5.4 mini and GPT-5.4 nano do not need a high-context tier: their official
maximum input is 272,000 tokens (400,000 total context including up to 128,000
output tokens), so they cannot satisfy the strict `input_tokens > 272000`
condition. They continue to use their base price.

Additional sources:

- <https://developers.openai.com/api/docs/models/gpt-5.4-mini>
- <https://developers.openai.com/api/docs/models/gpt-5.4-nano>
