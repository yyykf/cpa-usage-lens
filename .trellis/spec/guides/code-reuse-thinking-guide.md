# Code-Reuse Thinking Guide

## Trigger

Use before creating or duplicating a helper, formatter, component primitive,
DTO mapping, report aggregate, token/cost rule, query, constant, or error path.

## Search Current Owners

```bash
rg -n "costCoverage" backend frontend supabase .trellis/spec
```

Replace `costCoverage` with the real field, symbol, value, or behavior term for
the current change.

Check likely owners first:

- backend aggregation/DTO assembly: `backend/internal/report/report.go`;
- token-to-cost rules: `backend/internal/pricing/cost.go`;
- CPA normalization/replay: `backend/internal/collector/`;
- date ranges: `backend/internal/timeutil/`;
- API/auth request handling: `frontend/src/lib/api.ts`;
- formatting: `frontend/src/lib/format.ts`;
- canonical token segments: `frontend/src/lib/tokens.ts`;
- chart/data colors: `frontend/src/lib/charts.ts`;
- dashboard/UI primitives: `frontend/src/components/dashboard/` and
  `frontend/src/components/ui/`.

## Questions

- [ ] Is the proposed function already present under a different name?
- [ ] Is this one business rule with several consumers, or only similar-looking
      code with different ownership?
- [ ] Can the current owner accept one small extension without parameter sprawl?
- [ ] Would extraction reduce drift in two or more active consumers?
- [ ] Does the abstraction hide a meaningful distinction such as source versus
      key dimension, complete versus partial, or null versus zero?
- [ ] Will tests exercise the shared rule once and consumer-specific behavior
      separately?

## Project Rules to Protect

- Account/key reports reuse the same token and cost aggregation while keeping
  distinct dimension identity.
- All token composition uses `tokenSegments`; do not create per-component
  provider or legacy-field arithmetic.
- All cost strings use `formatCost`; do not create local null/zero rules.
- All API calls use the shared request boundary; do not duplicate token/401
  handling.
- Similar dashboard layout uses existing primitives before new wrappers.

## Do Not Abstract Yet

- One short local transformation with one consumer.
- Components that look similar but have different domain identity or interaction.
- A generic helper whose options exceed its actual logic.
- Future providers, pages, or pricing tiers with no present contract.

This preserves DRY without violating KISS/YAGNI.

## After Editing

- [ ] Search the old field/value/helper name again; no missed consumer remains.
- [ ] Confirm there is one authoritative implementation of the business rule.
- [ ] Confirm the shared abstraction has narrower or equal complexity.
- [ ] Run package quality gates from backend/frontend indexes.
