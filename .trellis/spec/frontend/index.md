# Frontend Engineering Contract

## Applies To

Read this index before changing `frontend/`, Go JSON DTOs consumed by the UI,
frontend-facing nginx behavior, or dashboard business semantics. These specs
describe the current React/Vite application, not generic React advice.

`MUST` and `MUST NOT` are review gates. `SHOULD` is the local default and needs
an explicit reason to diverge. `MAY` identifies supported choices.

## Architecture Invariants

- `pages/` owns page-level data orchestration. `components/` owns rendering and
  local interaction. `hooks/` owns reusable stateful behavior. `lib/` owns API
  access and pure reusable logic.
- Backend server state is currently fetched with the typed client in
  `src/lib/api.ts` and owned by `Dashboard`; no global server-state library is
  part of the architecture.
- `src/types.ts` mirrors Go JSON DTOs. A DTO change is a backend/frontend
  contract change even though this is a single repository.
- Token and cost displays use canonical fields and shared formatters. UI code
  MUST NOT re-infer provider semantics from legacy token fields.
- Tailwind v4 is CSS-first; theme and chart colors come from CSS variables.

## Change-Trigger Router

| Change trigger | Required specs |
| --- | --- |
| new page, component, hook, helper, or file move | [Directory Structure](./directory-structure.md) |
| props, composition, table/chart/empty/loading interaction | [Components](./component-guidelines.md), [Quality](./quality-guidelines.md) |
| reusable timer, storage-backed behavior, effect lifecycle | [Hooks](./hook-guidelines.md), [State Management](./state-management.md) |
| fetch sequence, refresh, auth/local/server/URL state | [State Management](./state-management.md), [Type Safety](./type-safety.md) |
| Go DTO, JSON field, nullable value, API response | [Type Safety](./type-safety.md), [Quality](./quality-guidelines.md) |
| token bucket, accounting quality, cost coverage, unknown/partial display | [Token Accounting Display](./token-accounting-display.md), [Type Safety](./type-safety.md) |
| Tailwind class, design token, chart color, shadcn primitive | [Styling](./styling-guidelines.md), [Components](./component-guidelines.md) |

## Pre-Development Checklist

- [ ] Read every spec selected by the router; this index is only navigation.
- [ ] Identify the current owner: page, feature component, dashboard primitive,
      UI primitive, hook, API client, or pure library.
- [ ] Search for the DTO field, formatter, chart color, token rule, and similar
      component before adding another implementation.
- [ ] Map `loading`, empty, error, `null`, zero, and valid-data states.
- [ ] For server data, map refresh concurrency and stale-response behavior.
- [ ] For token/cost changes, prove every view uses the same canonical meaning.
- [ ] Keep the change within existing dependencies unless the need for a new
      state or validation library is demonstrated by multiple real consumers.

## Quality Check

The current frontend has no unit-test or lint script. The real production gate
is TypeScript project build plus Vite bundle:

```bash
cd frontend
npm ci
npm run build
```

`npm run build` runs `tsc -b && vite build`. Do not claim a nonexistent test
command passed. For Go DTO changes also run backend tests; for styling changes
run the dist-CSS checks in [Styling](./styling-guidelines.md). Finish with
`git diff --check` at repository root.
