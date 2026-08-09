# Token Accounting Display Contract

> Read this before changing token composition, accounting quality, cost
> coverage, formatters, overview defaults, tables, trends, or model ranking.

## Scenario: Canonical six-bucket usage and honest cost coverage

### 1. Scope / Trigger

- Trigger: any change to Go/TypeScript token fields, `tokenSegments`,
  `EMPTY_OVERVIEW`, token bars, cost labels, quality counts, trend tooltips, or
  model/account/key ranking.
- CPA accounting v2 is canonical. The frontend displays validated backend
  semantics; it MUST NOT infer provider behavior from legacy aliases.
- Rationale lives in ADR
  [0003](../../../.project_context/design/decisions/0003-cpa-v2-accounting-quality.md).

### 2. Signatures

Canonical display buckets from `TokenBreakdown`:

```text
uncachedInputTokens
canonicalCacheReadTokens
cacheCreationTokens
nonReasoningOutputTokens
reasoningTokens
unclassifiedTokens
```

Authoritative invariant:

```text
sum(the six canonical buckets) = tokens
```

Accounting request counts:

```text
completeRequests
unclassifiedRequests
inconsistentRequests
legacyRequests
```

Cost contract:

```text
cost: number | null
costCoverage: complete | partial | unknown
```

### 3. Contracts

#### Six mutually exclusive buckets

- `frontend/src/lib/tokens.ts::tokenSegments` is the sole assembly point for
  display segments and fixed ordering.
- Components MUST use these canonical fields for token composition. Legacy
  `inputTokens`, `outputTokens`, `cachedTokens`, and `cacheReadTokens` remain
  diagnostic/compatibility fields, not additive display buckets.
- Never calculate `outputTokens + reasoningTokens`; reasoning is already a
  subset of output for protocols where the backend canonicalizes it.
- Never calculate `cachedTokens + cacheReadTokens`; CPA may publish the same
  cache read in both aliases.
- Overview, account table, key table, trend total, model daily/ranking total,
  and token composition MUST agree on authoritative `tokens` for the same
  range. The six-bucket UI MUST sum to that value.

#### Accounting quality

- Per request, `complete`, `unclassified`, and `inconsistent` are mutually
  exclusive quality states:
  - `complete`: canonical children reconcile with authoritative totals;
  - `unclassified`: total is known, but some tokens cannot be placed reliably;
  - `inconsistent`: declared accounting contradicts itself; the backend
    quarantines its authoritative total as unclassified rather than guessing.
- `legacyRequests` is orthogonal provenance, not a fourth quality state. A
  legacy request may also count as `completeRequests` after the provider-aware
  fallback. Therefore `legacyRequests` MUST NOT be added to the three quality
  counts as though all four were disjoint.

#### Cost coverage and visible unknown

- `costCoverage=complete`: all tokens in the range are reliably classified and
  required prices are available; show numeric cost, including `$0.0000`.
- `costCoverage=partial`: reliable classified tokens have a numeric cost while
  some tokens are unclassified/inconsistent; show the number plus `部分` and do
  not assign guessed cost to the remainder.
- `costCoverage=unknown`: `cost` is `null`; show `未知`, never zero.
- `unknown` is not expected to remain forever. It occurs when the range has no
  reliable classified billable tokens or a required model price is missing.
  Once reliable accounting and required prices exist, the API returns complete
  or partial numeric cost.
- Current DTOs do not distinguish “missing price” from “no reliable classified
  tokens.” UI copy MUST stay truthful for both; adding a reason requires a
  dedicated backend/frontend DTO change.
- `costCoverage` is aggregate cost usability. Do not confuse it with per-request
  accounting quality named `complete`.

#### Default and cross-view consistency

- `Dashboard.tsx::EMPTY_OVERVIEW` MUST contain every required DTO field. Its
  initial `costCoverage='unknown'` is a loading/default object, not proof that
  loaded data is unknown.
- `formatCost(null)` is the shared `未知` representation; numeric zero remains a
  number.
- Account/key tables sort null cost last and show `部分` for partial numeric
  cost. Trend tooltips and model ranking use the same meanings.
- `TokenComposition` shows accounting-quality diagnostics only after loading;
  it MUST not expose legacy aliases as extra token segments.

### 4. Validation & Error Matrix

| Input/state | Required display |
| --- | --- |
| six buckets sum to total, coverage complete, cost 0 | total matches; `$0.0000`; no unknown label |
| classified plus unclassified tokens, numeric cost | all six buckets shown; numeric cost plus `部分` |
| all tokens unclassified/inconsistent | total retained in unclassified; cost `未知` |
| complete accounting but required model price missing | token buckets still shown; cost `未知` |
| cost null in account/key/model/trend | `未知`; null sorts after known cost |
| `legacyRequests > 0` with complete fallback | may overlap `completeRequests`; no extra token bucket |
| loading with `EMPTY_OVERVIEW` | skeleton/loading treatment; do not announce a loaded unknown result |
| canonical sum differs from `tokens` | contract violation; fix upstream/data mapping, do not hide with normalization |

### 5. Good / Base / Bad Cases

- Good: total 120 = input 60 + cache read 30 + cache write 10 + ordinary
  output 5 + reasoning 15 + unclassified 0; complete numeric cost.
- Good: total 120 with 20 unclassified keeps total 120 and displays known cost
  for 100 classified tokens as `部分`.
- Base: a legacy request is normalized by backend, counted in both complete and
  legacy provenance, and displayed through the same six buckets.
- Bad: showing `inputTokens + cachedTokens + outputTokens + reasoningTokens`.
- Bad: changing `null` cost to `$0.0000` so a chart can render.
- Bad: turning an otherwise valid range into unknown only because one request
  has unclassified tokens; this loses the useful partial cost.

### 6. Tests Required

- Backend collector/report tests prove complete, unclassified, inconsistent,
  legacy, partial, all-unclassified, and missing-price cases.
- Backend JSON tests prove zero-valued canonical and quality fields remain in
  the response.
- Frontend production build proves DTO/default/component lockstep.
- For display behavior changes, verify overview, token composition, account,
  key, trend tooltip, and model rank with complete, partial, unknown, and zero.
- Search assertion: `tokenSegments` remains the only six-bucket assembly path.

```bash
rg -n "tokenSegments|uncachedInputTokens|canonicalCacheReadTokens|nonReasoningOutputTokens" frontend/src
cd backend && go test ./internal/collector ./internal/report ./internal/model
cd ../frontend && npm run build
```

### 7. Wrong vs Correct

#### Wrong

```ts
const total = row.inputTokens + row.cachedTokens + row.cacheReadTokens +
  row.outputTokens + row.reasoningTokens
const cost = row.cost ?? 0
```

This double-counts overlapping parent/alias fields and converts unknown cost to
a false known zero.

#### Correct

```ts
const segments = tokenSegments(row)
const displayedTotal = segments.reduce((sum, segment) => sum + segment.value, 0)
const displayedCost = formatCost(row.cost)
```

Assert `displayedTotal === row.tokens`; keep `row.cost === null` visible as
unknown and use `costCoverage` to distinguish complete from partial numeric cost.
