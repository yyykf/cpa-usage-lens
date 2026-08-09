# Frontend Directory and Ownership

## Trigger

Apply this spec when adding, moving, splitting, or reusing frontend code.

## Current Layout

```text
frontend/src/
├── pages/                  page composition and server-data orchestration
├── components/             feature-level dashboard views and interactions
│   ├── dashboard/          project-shared dashboard primitives
│   └── ui/                 shadcn/Radix UI primitives
├── hooks/                  reusable stateful React behavior
├── lib/                    API client and pure reusable calculations/formatters
├── types.ts                backend JSON DTO mirror
├── App.tsx                 auth-level route choice
├── main.tsx                React root bootstrap
└── index.css               Tailwind v4 theme and global styles
```

## Pattern

### Pages

- `pages/Dashboard.tsx` is the current owner of dashboard period, all server
  responses, loading lifecycle, refresh coordination, and logout action.
- A page MAY compose several feature components, but SHOULD pass already typed
  data and callbacks. It MUST NOT duplicate rendering primitives or token math.

### Components

- Root `components/*.tsx` files own feature panels such as account/key tables,
  trend, period selection, collector health, and model views.
- `components/dashboard/` owns project-specific reusable visuals such as
  `Panel`, `TokenBar`, `Empty`, `TableSkeleton`, and `StatRail`.
- `components/ui/` owns shadcn/Radix primitives. Domain fetching, accounting,
  pricing, and dashboard-specific labels MUST NOT be added there.
- Split a component when an independently named UI responsibility or repeated
  primitive emerges; do not create one-line wrapper files without reuse.

### Hooks and libraries

- Put reusable effect/state lifecycle in `hooks/`; `useAutoRefresh` is the
  reference timer/persistence hook.
- Put pure formatting, token segment creation, chart constants, delta math, and
  class merging in `lib/`.
- `lib/api.ts` is the only fetch/auth-token boundary. Components MUST NOT build
  ad hoc fetch calls or duplicate 401 handling.
- API DTOs shared by views belong in `types.ts`; component-only prop and UI
  union types stay beside the component.

### Imports and naming

- Use `@/*` for cross-directory imports under `src`. Short same-feature
  relative imports MAY be used where already established.
- React components use PascalCase files/symbols, hooks use `use*`, and library
  modules use lower-case domain names.
- Prefer named types for shared contracts and explicit `type` imports.

Evidence: `frontend/src/pages/Dashboard.tsx`, `frontend/src/lib/api.ts`,
`frontend/src/lib/tokens.ts`, `frontend/src/hooks/useAutoRefresh.ts`, and
`frontend/vite.config.ts`.

## Avoid

- Fetching in table/chart/UI primitive components.
- Provider-specific token math inside a component.
- A new `utils.ts` for logic already owned by `format.ts`, `tokens.ts`,
  `charts.ts`, `delta.ts`, or `api.ts`.
- Domain behavior inside `components/ui/`.
- A global state directory without a demonstrated cross-page requirement.

## Verify

```bash
rg -n "fetch\(" frontend/src
rg -n "uncachedInputTokens|canonicalCacheReadTokens|nonReasoningOutputTokens" frontend/src
cd frontend && npm run build
```

Review matches: network calls should remain in the API client, and canonical
token assembly should remain centralized in `lib/tokens.ts`.
