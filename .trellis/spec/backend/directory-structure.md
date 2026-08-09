# Backend Directory and Ownership

## Trigger

Apply this spec when adding a package, moving behavior, changing constructors or
interfaces, or wiring a new runtime service.

## Current Layout

```text
backend/
├── cmd/server/              process composition, goroutines, shutdown
├── internal/api/            HTTP routes, auth boundary, response DTO assembly
├── internal/collector/      CPA destructive queue, sanitization, replay, ingest
├── internal/config/         environment loading, defaults, validation
├── internal/db/             pgx queries, writes, transactions, state
├── internal/model/          cross-backend domain types and JSON DTOs
├── internal/pricing/        LiteLLM prices and token-to-cost semantics
├── internal/report/         daily rows plus prices into API reports
├── internal/rollup/         hot-to-daily scheduling and cleanup order
└── internal/timeutil/       configured-timezone calendar boundaries
```

## Pattern

### Composition boundary

- `cmd/server/main.go` MUST construct concrete dependencies, start long-running
  goroutines, install signal handling, and perform graceful shutdown.
- `main` MUST NOT absorb queue parsing, SQL, report calculations, or HTTP
  business rules. Put those in the owning `internal` package and inject them.
- Background loops MUST accept a `context.Context` and stop on cancellation.

Evidence: `backend/cmd/server/main.go`, `pricing.Service.RunDaily`,
`collector.Collector.Run`, and `rollup.Scheduler.Run`.

### Package ownership

- A package MUST own one stable concern shown above. Extend the existing owner
  before creating a parallel `service`, `util`, or `common` package.
- Cross-package data contracts belong in `internal/model`; package-private
  parsing or intermediate types stay with their owner.
- `internal/timeutil` is the single owner of day/range calculations. Handlers,
  reports, and rollups MUST NOT independently calculate calendar boundaries.
- Cost calculation belongs in `pricing`; grouping and DTO construction belong
  in `report`; SQL remains in `db`.

### Interfaces and dependencies

- Consumer packages SHOULD define the smallest interface they need. Existing
  examples are `api.DataStore`, `api.Prices`, `collector.Store`, and
  `rollup.Store`.
- Interfaces MUST NOT expose `*db.DB`, `pgxpool.Pool`, or unrelated methods only
  to make a concrete type fit.
- Constructors SHOULD receive collaborators and immutable configuration. Tests
  then use small fakes without a global container.

### Naming

- Package names are short, lower-case domain names: `collector`, `pricing`,
  `report`, not Java-style nested module names.
- Export a symbol only when another package consumes it. Keep helpers and
  transport-only types unexported.
- Tests live beside their package as `*_test.go`; database integration tests use
  the `integration` build tag.

## Avoid

- Handler code opening DB connections or reading environment variables.
- DB code importing API DTO behavior or frontend concepts.
- A second token normalization helper outside collector/pricing/report owners.
- Generic shared packages created for one caller.
- Package-level mutable singletons for DB, prices, clock, or auth.

## Verify

```bash
go list ./...
go test ./... -race
```

Review new imports and interfaces manually: dependency direction must follow the
current composition path, and no package may bypass its owning boundary.
