# Backend Logging and Secret-Safety Contract

## Trigger

Apply this spec when changing logs, collector state, authentication, CPA
payload handling, replay files, remote refreshes, or operator diagnostics.

## Pattern

- The backend currently uses Go's standard `log` package. New code SHOULD use
  the same mechanism unless a repository-wide observability change is approved.
- Log at lifecycle and handling boundaries: startup/shutdown, service mode,
  completed cleanup, refresh/recovery outcome, and failures requiring action.
- Prefix subsystem messages consistently with existing terms such as
  `价格：` and `rollup：` when the caller would otherwise be ambiguous.
- A failure log MUST name the operation and safe identifiers needed to act on
  it. Avoid dumping an entire object to gain context.
- Durable collector problems MUST also update `collector_state` when they affect
  health. Logs are history; `GET /api/collector` is current operator state.
- Resolved stale collector errors remain in DB for investigation but MUST NOT be
  returned as the current API error once collector status is healthy.

Evidence: `backend/cmd/server/main.go`, `backend/internal/api/handlers.go`,
`backend/internal/rollup/rollup.go`, and collector recovery tests.

## Secret Exclusion

The following MUST NOT appear in logs, DB rows, replay diagnostics, or API error
responses:

- `CPA_MANAGEMENT_KEY`, `DATABASE_URL`, `AUTH_TOKEN_SECRET`, dashboard password;
- plaintext client `api_key` or bearer/JWT token;
- raw CPA `response_headers`;
- raw `fail.body`;
- unsanitized CPA item or whole queue response.

Collector storage may retain only the sanitized replay contract. Key identity is
the SHA-256 fingerprint plus safe mask; logging plaintext first and sanitizing
later is forbidden. See [Usage Queue Contract](./usage-queue-contract.md).

## Severity Semantics

The standard logger has no structured level field. Express severity through
behavior and wording, not invented `INFO`/`WARN` strings:

- normal lifecycle: concise one-time message;
- recoverable degradation: operation plus error, keep safe fallback/retry;
- current collector failure: log and persist `last_error`/`last_error_at`;
- fatal prerequisite: `log.Fatalf` before the server becomes usable.

Do not add per-event success logs to the hot path. Aggregate counts and state are
the existing observability model.

## Avoid

- `log.Printf("payload=%+v", item)` or any raw JSON dump.
- Logging environment maps, request headers, authorization values, or URLs with
  embedded credentials.
- Logging every poll, row, or retry when state did not change.
- A log-only error for a failure that the API or collector-state contract must
  expose.
- Claiming exactly-once or zero-loss semantics in messages.

## Verify

```bash
rg -n "api_key|response_headers|fail\.body|Authorization|DATABASE_URL|AUTH_TOKEN_SECRET" backend
go test ./internal/collector/... ./internal/api/... -race
```

Review every match in context. Field names in sanitization/tests are expected;
secret values and raw-payload logging are not.
