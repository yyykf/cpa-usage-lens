# Component Contract

## Trigger

Apply this spec to React components, props, list rendering, tables, charts,
interactive controls, and dashboard primitives.

## Pattern

### Typed, presentation-focused components

- Every component MUST have typed props. Shared backend DTOs come from
  `src/types.ts`; component-only UI types remain local.
- Feature components receive data, loading state, and callbacks. Page-level
  fetching and refresh coordination stay in `pages/Dashboard.tsx`.
- Reuse `Panel`, `PanelHeader`, `Empty`, `TableSkeleton`, `TokenBar`, shared
  formatters, and chart constants before creating visual duplicates.

Evidence: `AccountTable.tsx`, `KeyTable.tsx`, `TrendChart.tsx`, and
`components/dashboard/Primitives.tsx`.

### Explicit view states

- Loading, empty, error/unknown, and valid zero are different states and MUST
  render intentionally.
- Tables/charts use their skeleton while loading and `Empty` only after loading
  completes with no rows.
- `null` cost means unknown and MUST display `未知`; numeric `0` is a known zero
  and MUST be formatted as cost.
- Collector nullable timestamps/lag use a neutral dash, not epoch or zero.

### Immutable derived views

- Sorting or transforming prop arrays MUST copy first (`[...items]`) and MAY use
  `useMemo` when calculation depends only on props/local sort state.
- React keys MUST use stable domain identity: source, model, full key
  fingerprint, date, or fixed segment key. Array index is forbidden where rows
  can reorder or change.
- Tie-breaks SHOULD be deterministic. Model ranking uses token/cost then model
  name; unknown cost sorts last.

### Interaction and accessibility

- Use real `<button>` elements or the shared `Button`; all buttons MUST set
  `type="button"` unless submitting a form intentionally.
- Controls need a visible/accessible label and disabled state during an active
  operation when duplicate actions would be unsafe.
- Decorative icons/marks SHOULD use `aria-hidden`; form inputs MUST have labels.
- Popover draft state MUST not update applied page state until the user confirms.

Evidence: `PeriodSwitcher.tsx`, `CollectorHealth.tsx`,
`ModelUsagePanel.tsx`, account/key sort headers, and `Login.tsx`.

## Avoid

- Clickable `<div>`/`<span>` elements.
- Mutating an array received through props.
- Formatting numbers/cost/dates inline when `lib/format.ts` owns the rule.
- Recomputing canonical token segments outside `lib/tokens.ts`.
- Using `cost || 0` or `cost ?? 0` for display.
- Copying shadcn primitive code into feature components.

## Verify

- Exercise loading, empty, zero, `null`, and populated states for every changed
  panel.
- Check keyboard focus and disabled behavior for changed controls.
- Run `npm run build`.
