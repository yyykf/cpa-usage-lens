# Spec Evidence Map

This map records why each project spec exists and which repository evidence
supports it. It is task research, not a second normative source.

## Backend

| Spec | Rules to preserve | Primary evidence | Drift to prevent |
| --- | --- | --- | --- |
| `backend/directory-structure.md` | `cmd/server` owns composition/lifecycle; `internal` packages own one concern; API and schedulers depend on small consumer-owned interfaces | `backend/cmd/server/main.go`, `backend/internal/api/server.go`, `backend/internal/rollup/rollup.go` | handlers constructing infrastructure, package cycles, broad shared interfaces |
| `backend/database-guidelines.md` | pgx only; parameterized SQL; batch insert; query/scan lockstep; append-only idempotent migrations; hot is source fact and daily is derived | `backend/internal/db/*.go`, `supabase/migrations/*.sql`, `backend/internal/db/rollup_integration_test.go` | ORM introduction, SQL interpolation, edited released migrations, daily-only facts |
| `backend/error-handling.md` | return wrapped internal errors; log details at handling boundary; stable JSON errors; fail-closed destructive-source behavior | `backend/internal/api/handlers.go`, `backend/internal/db/db.go`, `backend/internal/collector/collector.go` | leaking internals, double logging, silent collector loss |
| `backend/logging-guidelines.md` | standard `log`; actionable lifecycle/state messages; no secrets or raw CPA payloads | `backend/cmd/server/main.go`, `backend/internal/collector/sanitize.go`, `backend/internal/collector/collector.go` | credential leakage, noisy per-row logs, hidden recovery failures |
| `backend/quality-guidelines.md` | `gofmt`, build, vet, race tests; table-driven/local fake tests; integration gate for migrations | `.github/workflows/ci.yml`, backend tests, `backend/internal/db/rollup_integration_test.go` | commands not matching CI, untested concurrency, fake migration confidence |
| `backend/usage-queue-contract.md` | destructive pop, sanitized durable replay before typed decode, at-least-once, composite DB idempotency | `backend/internal/collector/replay.go`, `buffer.go`, `collector.go`, ADR 0001/0004 | parse-before-persist loss, secret-bearing buffers, multiple collectors |
| `backend/cost-calculation.md` | provider-aware legacy fallback; canonical v2; query-time cost; partial versus unknown; long-context per-request tier | `backend/internal/pricing/cost.go`, `backend/internal/report/report.go`, ADR 0002/0003 | cache/reasoning double count, guessed cost, persisted stale cost |
| `backend/deployment-guidelines.md` | production proxy topology; debug override; breaking migration order | Compose files, `frontend/nginx.conf`, `.github/workflows/release.yml` | exposed backend, stale release commands, old binary/new schema overlap |

## Frontend

| Spec | Rules to preserve | Primary evidence | Drift to prevent |
| --- | --- | --- | --- |
| `frontend/directory-structure.md` | pages orchestrate; components render/local-interact; `components/ui` owns shadcn primitives; hooks and lib have narrow roles | `frontend/src/pages/Dashboard.tsx`, `frontend/src/components/**`, `frontend/src/lib/**` | data orchestration in primitives, generic helpers in components |
| `frontend/component-guidelines.md` | typed props; semantic controls; explicit loading/empty/data states; immutable prop sorting; stable keys | account/key/model tables, dashboard primitives, `PeriodSwitcher.tsx` | mutating props, index keys, clickable divs, mixed states |
| `frontend/hook-guidelines.md` | reusable stateful behavior only; latest callback ref; timer cleanup; finite persisted values | `frontend/src/hooks/useAutoRefresh.ts`, `Dashboard.tsx` | leaked intervals, stale closures, unvalidated storage |
| `frontend/state-management.md` | page-owned server state; parallel refresh; request sequence; silent in-flight lock/error dedupe; local/auth preference boundaries | `Dashboard.tsx`, `frontend/src/lib/api.ts`, `useAutoRefresh.ts` | stale response overwrite, polling pile-up, unnecessary global store |
| `frontend/type-safety.md` | strict TypeScript; DTO mirror in `types.ts`; no hidden zero fields; null means semantic unknown; runtime validation at existing boundaries | `frontend/tsconfig.json`, `frontend/src/types.ts`, `backend/internal/model/types.go` | frontend/backend field drift, `null` coerced to zero, untyped API data |
| `frontend/token-accounting-display.md` | canonical six mutually exclusive buckets; total invariant; quality and cost coverage semantics; all views agree | `frontend/src/lib/tokens.ts`, token/table/chart components, backend model/report, ADR 0003 | legacy-field double addition, unknown shown as zero, per-view semantics |
| `frontend/styling-guidelines.md` | Tailwind v4 CSS-first; shadcn HSL tokens; `@theme inline`; semantic chart colors | `frontend/src/index.css`, `vite.config.ts`, `frontend/src/lib/charts.ts` | hard-coded colors, v3 config reintroduction, theme indirection drift |
| `frontend/quality-guidelines.md` | production build is current gate; no invented frontend test command; API proxy and SPA image must still build | `frontend/package.json`, `.github/workflows/ci.yml`, `frontend/Dockerfile`, `nginx.conf` | claims of nonexistent tests, type errors, dev-only success |

## Shared Guides

| Guide | Project-specific trigger | Evidence |
| --- | --- | --- |
| `guides/cross-layer-thinking-guide.md` | DTO/schema/token/accounting/storage/report/display changes | migrations, Go DTOs, TypeScript DTOs, report builders, display components |
| `guides/code-reuse-thinking-guide.md` | repeated report aggregation, token/cost display, formatting, API request, dashboard primitives | `report.go`, `lib/tokens.ts`, `lib/format.ts`, `lib/api.ts`, `components/dashboard/` |

## Known Current Constraint

`costCoverage=unknown` currently has two causes: no reliable classified token
cost, or a required model price is missing. The API does not expose a separate
reason code. Specs must preserve honest `null`/unknown display and must not claim
the frontend can distinguish these causes until a dedicated product change
extends the DTO.
