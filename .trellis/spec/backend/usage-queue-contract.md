# CPA Usage-Queue Consumption Contract

> Read this before changing `internal/collector`, CPA queue parsing, disk
> buffering, event identity, or collector deployment topology.

Decision context: ADR
[0001](../../../.project_context/design/decisions/0001-usage-hot-composite-pk.md)
defines replay identity, and ADR
[0004](../../../.project_context/design/decisions/0004-durable-sanitized-usage-replay.md)
defines persist-before-parse replay.

## Scenario: Durable consumption of destructive usage events

### 1. Scope / Trigger

- Trigger: any change to CPA management HTTP calls, queue payload fields,
  secret sanitization, replay files, `toEvent`, database insertion, collector
  recovery, or replica count.
- `GET /v0/management/usage-queue` is a destructive read. Items returned by CPA
  are removed before Lens can acknowledge, validate, or retry them.
- The queue has finite configurable retention. Verify the live CPA value during
  deployment; never assume an expired source event can be recovered by Lens.

### 2. Signatures

CPA transport boundary:

```text
CPAClient.PopUsageRaw(ctx, count) -> []json.RawMessage
```

The client may split the HTTP response into array items, but MUST NOT perform
field-level decoding before durable buffering.

Replay file envelope (`schema_version=1`):

```json
{
  "schema_version": 1,
  "popped_at": "RFC3339 timestamp",
  "items": [
    {
      "payload": {},
      "key_fingerprint": "sha256 hex or none",
      "key_mask": "safe display mask",
      "sanitization_error": "optional non-secret diagnostic"
    }
  ]
}
```

`payload` retains the sanitized CPA item, including unknown fields, legacy
`tokens`, `accounting_version`, and `token_breakdown`. It MUST NOT contain:

```text
api_key
response_headers
fail.body
```

Legacy read compatibility:

```text
top-level JSON array -> legacy []model.UsageEvent buffer
top-level JSON object -> versioned replay envelope
```

New code writes only the versioned envelope.

### 3. Contracts

#### Mandatory order

```text
destructive pop
-> split into raw JSON items
-> derive key fingerprint/mask and remove sensitive fields
-> write same-directory temporary file
-> file fsync
-> atomic rename
-> directory fsync where supported
-> typed decode and accounting validation
-> idempotent DB insert
-> DB confirmation
-> delete or retain the buffer artifact
```

- The only field-level work allowed before durable save is deterministic secret
  sanitization and replay-envelope construction.
- Raw in-memory messages containing plaintext keys MUST be cleared immediately
  after sanitized replay items are built.
- `toEvent` and accounting validation MUST run after `SaveReplayBatch` on the
  normal collector path.
- Normal polling and startup recovery MUST share the same replay-to-event and
  DB insertion path.

#### Delivery semantics

- Delivery is at-least-once, not exactly-once.
- A crash after DB success but before buffer deletion replays the same physical
  event. The DB key `(request_id, event_ts, total_tokens)` and `ON CONFLICT DO
  NOTHING` MUST keep this replay idempotent.
- Never replace the composite event key without proving both multi-turn event
  separation and buffer-replay idempotency.
- Run exactly one collector against one CPA queue. Multiple collectors split the
  destructive stream rather than providing safe parallelism.

#### Rejected and corrupt artifacts

- A sanitized item that cannot be strongly decoded, lacks required identity or
  time, or otherwise cannot become a `UsageEvent` is counted as rejected.
- Valid items in the same batch are inserted. After DB confirmation, the
  sanitized source batch is renamed from `.json` to `.json.rejected`, preserving
  evidence while excluding it from automatic replay.
- Malformed declared-v2 accounting that can be decoded is not rejected. It
  follows the accounting contract and becomes v2 `inconsistent`.
- Invalid JSON files and unsupported replay schema versions are renamed to
  `.json.corrupt`; the collector records an observable error.
- Commit, rejection, quarantine, and recovery failures MUST NOT be ignored.
  Where they affect collector recovery, record them in collector state as well
  as logs.

#### Residual limits

Lens cannot close every source-side window:

- process termination after CPA removes items but before the sanitized envelope
  rename completes can still lose those items;
- a malformed whole HTTP response that cannot be split into JSON array items
  cannot be safely persisted because it may still contain secrets;
- source events expired before Lens polls them are unrecoverable.

The design minimizes these windows; it MUST NOT claim exactly-once delivery or
zero possible loss.

### 4. Validation & Error Matrix

| Condition | Required behavior |
| --- | --- |
| Empty queue | Update poll health; do not create a buffer file |
| Valid CPA item | Sanitize, persist, decode, insert, then delete buffer |
| CPA field type drift in one item | Persist sanitized raw item first; insert valid siblings; retain batch as `.rejected` |
| Missing request ID or invalid timestamp | Retain sanitized batch as `.rejected`; never silently skip |
| Malformed declared-v2 breakdown | Store as accounting v2 `inconsistent`, not legacy and not rejected |
| DB insert failure | Keep `.json` pending for startup recovery |
| Crash after DB success | Replay safely; DB conflict key prevents duplicate row |
| Buffer save failure | Attempt best-effort processing and emit a data-loss-risk collector error |
| Directory fsync failure after rename | Return both the handle and error so the renamed file can still be processed safely |
| Legacy `[]UsageEvent` buffer | Insert through the shared DB path, then commit |
| Unsupported envelope version | Rename to `.corrupt`; log and update collector state |
| Corrupt buffer JSON | Rename to `.corrupt`; log and update collector state |
| Buffer commit/reject failure | Keep/replay the `.json` file and surface the failure |
| Second collector instance | Forbidden; stop before it consumes part of the stream |

### 5. Good / Base / Bad Cases

- Good: a CPA item contains a future field and a token field with an unexpected
  type. Lens removes secrets, durably keeps both fields as raw JSON, then marks
  only the unparseable item rejected after save.
- Good: DB insertion succeeds but the process dies before file deletion. Startup
  replay reaches the same composite key and inserts zero duplicate rows.
- Base: a pre-v0.6.1 `[]UsageEvent` buffer remains readable after upgrade.
- Bad: decoding directly into `[]rawQueueItem` inside the HTTP client. One field
  type drift can discard a whole already-popped batch before buffering.
- Bad: storing CPA JSON unchanged. It can persist API keys, response headers, and
  failure bodies on disk.
- Bad: deleting a buffer before DB confirmation or ignoring a commit error.
- Bad: calling usage-queue for health checks, debugging, or release acceptance.

### 6. Tests Required

- HTTP client test: return per-item `json.RawMessage` even when a field has an
  unexpected type.
- Sanitization test: persisted JSON contains no plaintext key,
  `response_headers`, or `fail.body`; unknown and accounting fields survive.
- Memory hygiene test: raw secret-bearing messages are cleared after envelope
  construction.
- Durability test: save uses temp file, file sync, rename, and directory sync;
  a post-rename directory-sync error retains the file handle.
- Round-trip test: valid accounting v2 produces the same canonical event before
  and after buffer replay.
- Collector-order test: a pending `.json` file exists when DB insertion begins.
- Partial-batch test: valid siblings insert while invalid timestamp/type-drift
  items remain in a sanitized `.rejected` artifact.
- Recovery test: DB failure leaves a pending replay file; startup recovery later
  inserts and commits it.
- Compatibility test: legacy `[]UsageEvent` buffer files still recover.
- Quarantine test: corrupt/unsupported files become `.corrupt` and update
  collector state.
- Full gates: `go test ./...`, `go test -race ./...`, `go vet ./...`, formatting,
  frontend production build, and `git diff --check`.

### 7. Wrong vs Correct

#### Wrong

```go
items, err := client.PopUsage(ctx, count) // []rawQueueItem: typed decode first
for _, item := range items {
    event, ok := toEvent(item)
    if ok {
        events = append(events, event)
    }
}
handle, err := buffer.Save(events)
```

Typed decode, validation, and rejection happen after destructive pop but before
durable save. A crash or field-type drift loses data.

#### Correct

```go
items, err := client.PopUsageRaw(ctx, count) // []json.RawMessage
batch := newReplayBatchFromRaw(items, poppedAt) // sanitize and clear secrets
handle, saveErr := buffer.SaveReplayBatch(batch)
result, processErr := collector.processPending(ctx, handle, pendingBatch{Replay: batch})
```

The sanitized protocol evidence is durable before typed decoding and accounting
validation. DB confirmation then decides whether the file is committed or kept
for recovery/diagnosis.
