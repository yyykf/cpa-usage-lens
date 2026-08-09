# Frontend Type and DTO Contract

## Trigger

Apply this spec to TypeScript types, API response fields, nullability, JSON DTOs,
localStorage parsing, chart payloads, and type assertions.

## Compile-Time Contract

- TypeScript remains `strict` with `isolatedModules` and
  `noFallthroughCasesInSwitch`. Do not weaken `frontend/tsconfig.json` to make a
  change compile.
- Shared API DTOs live in `frontend/src/types.ts` and MUST match JSON names and
  nullability in `backend/internal/model/types.go`.
- Type-only imports SHOULD use `import type`.
- Finite protocol/state values use literal unions, for example `Period`,
  `ModelMetric`, `CostCoverage`, collector status, and refresh intervals.
- Component props and non-library callback results MUST be typed; avoid implicit
  object shapes that duplicate a shared DTO.

## Backend-to-Frontend DTO Lockstep

For every backend JSON DTO change:

1. update Go DTO fields/tags and their zero-value serialization test;
2. update `frontend/src/types.ts`;
3. update all default/empty objects such as `EMPTY_OVERVIEW`;
4. search every consumer and preserve null/zero/empty semantics;
5. run backend tests and frontend production build.

Required numeric/count/quality fields MUST remain serialized at zero. Adding
`omitempty` creates runtime `undefined` while the TypeScript contract says
`number` or a required union.

## Null and Runtime Boundaries

- `number | null` is a business contract, not optional typing noise. Cost
  `null` means unknown; zero means a known zero cost.
- Nullable collector timestamps and lag remain `null` until known. Do not use
  empty string, epoch, or zero as a substitute.
- API fetch currently trusts same-repository JSON with a generic response type;
  compile-time typing is not runtime validation. New untrusted/external payloads
  MUST be validated at their boundary before entering typed state.
- localStorage values are untrusted strings and MUST be parsed and checked
  against allowed values before assertion.
- A narrow chart-library adapter type MAY be local to the chart; do not use
  broad `any` to bypass third-party payload typing.

Evidence: `frontend/src/types.ts`, `frontend/src/lib/api.ts`,
`frontend/src/hooks/useAutoRefresh.ts`, `TrendChart.tsx`, and
`backend/internal/model/types_test.go`.

## Avoid

- `any`, `@ts-ignore`, or `as unknown as ...` in application code.
- Treating a missing required field as zero in each component.
- Making a DTO field optional only to avoid updating `EMPTY_OVERVIEW`.
- `cost ?? 0`, `cost || 0`, or formatting null through a numeric formatter.
- Maintaining a second copy of token/accounting field types inside components.

## Verify

```bash
cd backend && go test ./internal/model ./internal/report
cd ../frontend && npm run build
```

Search the changed field name across Go, SQL, TypeScript, defaults, and display
components before review.
