# Backend Error Contract

## Trigger

Apply this spec when adding failure paths, HTTP responses, config validation,
database operations, background loops, queue replay, or recovery behavior.

## Pattern

### Return and wrap at lower layers

- A function that cannot complete its contract MUST return an error. Do not log
  and return success, and do not use zero values as hidden failure signals.
- Add operation context when crossing infrastructure boundaries. Preserve the
  original error with `%w` where callers may need the cause.
- Constructors MUST reject invalid mandatory state before starting goroutines.
  `config.Load`, `db.Open`, and `api.NewAuthenticator` are reference boundaries.
- Optional startup capabilities MAY fail without stopping the process only when
  a safe fallback exists. Example: the price service can continue from its DB
  cache when remote refresh fails.

### Handle once at the boundary

- HTTP handlers parse and validate input, call the owning report/store service,
  log internal detail once, and return stable JSON through `writeJSON`.
- Do not expose SQL, connection strings, JWT details, upstream payloads, file
  paths, or stack-like diagnostics to clients.
- Expected client failures use stable 4xx responses. Current examples:
  malformed login body `400`, bad credentials/token `401`, invalid period `400`.
- Unexpected infrastructure failures use a stable 500 message such as
  `{"error":"查询失败"}` while the server log keeps the actual cause.
- A helper that writes the response and returns `ok=false` MUST end the handler
  path immediately. Never write a second response.

Evidence: `backend/internal/api/server.go` and
`backend/internal/api/handlers.go`.

### Destructive-source and background failures

- After a destructive CPA pop, a save, decode, insert, commit, reject, or
  quarantine failure MUST remain observable. Follow
  [Usage Queue Contract](./usage-queue-contract.md); never silently drop an item.
- DB failure leaves a pending replay artifact. Decode rejection preserves a
  sanitized `.rejected` artifact. Corrupt or unsupported envelopes become
  `.corrupt`.
- Rollup failure MUST stop cleanup for that tick.
- Background loops SHOULD report the failing operation and continue only when
  retrying is safe. Fatal process prerequisites are handled in `main` before
  serving traffic.

## Validation Matrix

| Condition | Required result |
| --- | --- |
| missing required environment variable | startup fails with key name, not secret value |
| invalid request JSON | stable `400` JSON response |
| invalid/expired bearer token | stable `401`, protected handler not called |
| invalid period/range | stable `400`, no DB query |
| report query failure | detailed server log plus stable `500` JSON |
| price refresh failure | stable `500`; old cache remains usable |
| rollup failure | log error and skip deletion |
| collector DB failure | keep pending buffer and update observable error state |

## Avoid

- `panic` for request, queue, DB, or remote-data errors.
- `_ =` on material persistence, quarantine, commit, or cleanup operations.
- Returning `200` with an empty object after an internal failure.
- Logging the same error in every layer.
- Client messages assembled from `err.Error()`.
- Treating parser rejection as successful consumption without retained evidence.

## Verify

- Add a negative-path test at the layer that owns the response or recovery
  decision. Assert status/result and that forbidden side effects did not occur.
- Run `go test ./... -race`.
- For collector failures, assert the replay file state and collector-state error,
  not only the returned error.
