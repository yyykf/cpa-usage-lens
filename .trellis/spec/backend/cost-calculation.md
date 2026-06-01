# Cost Calculation Contract

> How token cost is computed in `internal/pricing/cost.go`. Read this BEFORE changing
> pricing logic — the cache semantics are counter-intuitive and have shipped a bug once.

---

## Provider cache semantics differ (the expensive trap)

Token usage is reported and billed differently per provider:

| | OpenAI / GPT | Anthropic / Claude |
|---|---|---|
| Is `cached` part of `input_tokens`? | **Yes** — `cached_tokens` is a *subset* of `input_tokens` | **No** — input counts non-cached only |
| Cache fields populated | `cached_tokens` | `cache_read_tokens` / `cache_creation_tokens` |
| Cache pricing | cached billed at the `cache_read` rate (≈ 1/10 of input for gpt-5.4) | each field has its own rate |

**The trap**: for OpenAI, charging the full `input_tokens` at input price
double-charges the cached portion at ~10× its real rate. Cache-heavy traffic
(Codex runs hit 80–99% cache) is overcharged by up to ~10×.

## The rule (auto-detect by field shape, no provider name)

```text
billableInput = input_tokens
if cached > 0 && cache_read == 0:          # OpenAI shape
    billableInput = input_tokens - cached   # split cached OUT of input
    cost += cached * cache_read_price        # cached at discount (fallback: input price)
cost += billableInput * input_price
# Claude shape (cached == 0): input stays whole; cache_read / cache_creation added separately
```

- Shape is detected by `cached > 0 && cache_read == 0`, NOT by a provider name.
  No strategy pattern yet — intentional (YAGNI).
- `reasoning_tokens` bill at the **output** price.
- If a token type is > 0 but its price is missing → return `unknown` (nil cost),
  never `$0`. Cost is computed at query time from the current price table, never
  stored, so price changes need no backfill.
- Defensive: if `cached > input` (bad data), clamp `billableInput` to 0; no negative cost.

## When to revisit (evolve to a strategy pattern)

If a future provider's cache semantics don't fit "cached ⊆ input  XOR  independent
cache_read/creation", the field-shape heuristic breaks. That is the signal to
introduce a per-provider strategy. Until then, keep it simple.

## Why this matters

This exact bug shipped once: total cost showed **$3.9975** when it should have been
**$2.2012** (−45%). The fix was ~8 lines; finding it required real data + manual
recompute. Any pricing change that ignores this contract silently re-introduces the
overcharge — there is no test that will scream unless you keep the cached-discount cases.
