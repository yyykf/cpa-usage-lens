# Frontend Quality Gates

## Trigger

Apply this spec to every frontend or backend DTO change before commit and
review.

## Required Practices

- Keep page orchestration, feature components, hooks, and pure libraries within
  their documented ownership boundaries.
- Reuse the API client, formatters, canonical token helper, chart constants,
  dashboard primitives, and shadcn primitives.
- Preserve strict null/zero/empty/loading semantics and deterministic ordering.
- Effects and timers MUST clean up; async refresh MUST prevent stale responses
  and polling overlap.
- User interactions use semantic controls, explicit button types, labels, focus
  states, and safe disabled states.
- Styles use theme tokens and existing responsive composition; no one-off raw
  colors for shared data semantics.

## Current Automated Gate

The repository intentionally has no frontend unit-test or lint script today.
Do not invent or report one. CI runs:

```bash
cd frontend
npm ci
npm run build
```

`npm run build` is both the TypeScript and production-bundle gate. A frontend
test runner MAY be introduced by a dedicated change with scripts, CI wiring,
and initial tests; until then, behavior-sensitive UI changes need explicit
manual/browser evidence in the task or PR.

## Change-Specific Verification

| Change | Required proof |
| --- | --- |
| Go/TypeScript DTO | backend DTO/report tests plus frontend build |
| token or cost display | [Token Accounting Display](./token-accounting-display.md) cases across affected views |
| timer/refresh | overlap, stale response, cleanup, silent error behavior |
| table/ranking | immutable sort, null-last cost, deterministic ties, stable keys |
| loading/empty behavior | loading, empty, zero, null, populated states |
| styling/theme | build and dist-CSS checks from [Styling](./styling-guidelines.md) |
| nginx/Docker | production image build or equivalent config validation |

## Avoid

- Passing `npm run build` while leaving wrong runtime null/empty semantics
  unreviewed.
- Mutating props, index keys on reorderable data, or ad hoc fetch calls.
- Adding `any` or compiler suppressions to adapt a third-party component.
- New direct dependencies for behavior already expressible by current helpers.
- Claiming browser/E2E verification that was not run.

## Review Checklist

- [ ] Directory owner and state owner are correct.
- [ ] Existing helper/primitive search completed.
- [ ] DTO/default objects and every consumer agree.
- [ ] Loading, empty, zero, null, error, and populated states are intentional.
- [ ] Async effects cannot leak, overlap, or apply stale state.
- [ ] `npm run build` and repository-root `git diff --check` pass.
