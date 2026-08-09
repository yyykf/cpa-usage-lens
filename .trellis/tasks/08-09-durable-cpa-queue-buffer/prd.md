# Durably Buffer Sanitized CPA Queue Payloads

## Goal

Close the data-loss window between CPA's destructive usage-queue pop and the
current normalized-event buffer. Persist a versioned, sanitized, replayable
queue envelope before timestamp/accounting validation, then process it with
at-least-once replay and the existing database idempotency key.

Target release: patch release `v0.6.1`, subject to live release revalidation.

## Why

CPA `GET /v0/management/usage-queue` removes returned items permanently. The
current collector calls `toEvent` before `Buffer.Save`; a crash, parser defect,
or rejected item in that interval cannot be replayed. The code therefore does
not satisfy the existing usage-queue contract's stated `pop -> persist -> DB`
order.

## Requirements

### Durable sanitized envelope

- Buffer a versioned batch envelope, not `[]model.UsageEvent`.
- Preserve every CPA field required for later normalization, including legacy
  tokens, `accounting_version`, and `token_breakdown`.
- Never persist plaintext `api_key`, response headers, or failure body.
- Before buffering, derive the existing key fingerprint and safe display mask,
  store those sanitized values, then clear the plaintext key.
- The only allowed work between pop and durable save is deterministic secret
  sanitization and construction of the replay envelope.
- Buffer writes must use a same-directory temporary file, file `fsync`, atomic
  rename, and best-effort directory `fsync` where supported.

### Processing and recovery

- Normalization and accounting validation happen only after the envelope is
  durably saved.
- Normal polling and startup recovery must use one shared envelope-processing
  path.
- A malformed declared-v2 accounting payload remains `inconsistent`; it must
  not become legacy or disappear.
- An item missing required identity/time fields must remain recoverable and
  observable. It must not be silently dropped after a successful pop.
- Unsupported envelope versions or corrupt files are quarantined and reported;
  they are never deleted as successfully committed.
- Database success precedes buffer deletion. A crash after DB success but before
  deletion may replay; the existing `(request_id, event_ts, total_tokens)` key
  must make replay idempotent.
- Buffer commit/quarantine failures must be logged and surfaced in collector
  state where they affect recovery; do not ignore them silently.

### Compatibility

- Support pending legacy buffer files containing `[]model.UsageEvent`, or ship
  an explicit safe one-time conversion. Preferred KISS path: read both formats,
  write only the new format.
- No database migration, API DTO change, frontend change, new environment
  variable, or new deployment parameter.
- Keep single-collector and destructive-pop constraints unchanged.

### Operations and release

- Update the usage-queue code-spec with the implemented envelope schema,
  failure matrix, replay semantics, and known residual window.
- Add an execution record for candidate and official vmrack validation.
- Build candidate images from the exact feature commit; deploy with collector
  disabled first; verify health and buffer compatibility; then enable one
  collector.
- Do not call the destructive usage-queue endpoint for validation.
- Open a PR against `main`, require green CI, merge, publish the patch release,
  and deploy official images after candidate acceptance.

## Acceptance Criteria

- [ ] After pop, sanitized raw queue envelopes are saved before `toEvent` or
      accounting validation runs.
- [ ] Buffer files contain no plaintext API key, response headers, or failure
      body, while retaining accounting v2 and legacy token fields.
- [ ] Startup recovery reconstructs the same `UsageEvent` and accounting result
      as the normal polling path.
- [ ] A DB failure keeps the envelope pending; retry inserts once without
      duplicate rows.
- [ ] A crash-equivalent replay after DB success is harmless through DB
      idempotency.
- [ ] Corrupt and unsupported envelope files are quarantined and observable.
- [ ] Legacy `[]UsageEvent` pending files still recover safely.
- [ ] Unit tests cover save/load/commit, redaction, v2 round-trip, legacy read,
      unsupported/corrupt input, DB failure, replay, and invalid item handling.
- [ ] `go test ./...`, `go test -race ./...`, `go vet ./...`, formatting checks,
      frontend production build, and `git diff --check` pass.
- [ ] vmrack candidate validation proves one collector, stable health/restarts,
      advancing collector waterline, no pending buffer accumulation, and no
      secret-bearing buffer files.
- [ ] PR merged, patch release published, official images deployed, and release
      runtime revalidated.

## Non-Goals

- Exactly-once delivery.
- Changing CPA itself or adding an acknowledgement protocol.
- Redesigning database primary keys.
- General spec bootstrap outside the usage-queue contract.
- Changing token accounting, pricing, report DTOs, or frontend behavior.

## Design Notes

- Delivery model: at-least-once replay plus database idempotency.
- Residual unavoidable window: process termination after CPA response is
  received but before sanitized envelope persistence finishes.
- Prefer a small versioned envelope owned by `internal/collector`; avoid a
  generic queue framework.
- Existing project ADR 0001 defines the database replay idempotency key.
